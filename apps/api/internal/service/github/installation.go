// Package github provides services for GitHub App functionality:
// installation lifecycle management and webhook event dispatching.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/oyetanishq/yappr/apps/shared/config"
	sharedgithub "github.com/oyetanishq/yappr/apps/shared/github"
	"github.com/oyetanishq/yappr/apps/shared/model"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const (
	reposCachePrefix = "github:repos:"
	reposCacheTTL    = 5 * time.Minute
)

// InstallationService manages GitHub App installation records in MongoDB.
type InstallationService struct {
	col      *mongo.Collection
	cfg      *config.Config
	log      *zap.Logger
	rdb      *redis.Client
	ghClient *sharedgithub.Client
}

// NewInstallationService creates the service and ensures required indexes exist.
func NewInstallationService(rdb *redis.Client, client *mongo.Client, cfg *config.Config, log *zap.Logger) (*InstallationService, error) {
	col := client.Database(cfg.Mongo.DB).Collection("installations")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Unique index on installation_id — GitHub guarantees uniqueness.
	if _, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "installation_id", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("installation_id_unique"),
	}); err != nil {
		log.Warn("mongo installations index: installation_id", zap.Error(err))
	}

	// Index on user_id for fast per-user listing.
	if _, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "user_id", Value: 1}},
		Options: options.Index().SetName("installations_user_id"),
	}); err != nil {
		log.Warn("mongo installations index: user_id", zap.Error(err))
	}

	ghClient := sharedgithub.NewClient(cfg.GithubApp.AppID, cfg.GithubApp.PrivateKey)

	return &InstallationService{col: col, cfg: cfg, log: log, rdb: rdb, ghClient: ghClient}, nil
}

// Save upserts an installation by installation_id.
// If the installation already exists (e.g. user re-installs), it updates the
// account_login and updated_at fields.
func (s *InstallationService) Save(ctx context.Context, installationID int64, userID, accountLogin string) (*model.Installation, error) {
	now := time.Now().UTC()

	filter := bson.D{{Key: "installation_id", Value: installationID}}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "user_id", Value: userID},
			{Key: "account_login", Value: accountLogin},
			{Key: "app_id", Value: s.cfg.GithubApp.AppID},
			{Key: "updated_at", Value: now},
		}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "_id", Value: uuid.NewString()},
			{Key: "installation_id", Value: installationID},
			{Key: "created_at", Value: now},
		}},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var inst model.Installation
	if err := s.col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&inst); err != nil {
		return nil, fmt.Errorf("installation: save: %w", err)
	}
	return &inst, nil
}

// ListForUser returns all installations belonging to the given userID.
func (s *InstallationService) ListForUser(ctx context.Context, userID string) ([]model.Installation, error) {
	cur, err := s.col.Find(ctx,
		bson.D{{Key: "user_id", Value: userID}},
		options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	)
	if err != nil {
		return nil, fmt.Errorf("installation: list: %w", err)
	}
	defer cur.Close(ctx)

	var installs []model.Installation
	if err := cur.All(ctx, &installs); err != nil {
		return nil, fmt.Errorf("installation: decode: %w", err)
	}
	if installs == nil {
		installs = []model.Installation{}
	}
	return installs, nil
}

// GetByInstallationID returns a single installation record by its GitHub installation_id.
func (s *InstallationService) GetByInstallationID(ctx context.Context, installationID int64) (*model.Installation, error) {
	var inst model.Installation
	err := s.col.FindOne(ctx, bson.D{{Key: "installation_id", Value: installationID}}).Decode(&inst)
	if err != nil {
		return nil, fmt.Errorf("installation: get by id: %w", err)
	}
	return &inst, nil
}

// ListRepos returns the repositories accessible to the given GitHub App installation.
// Results are cached in Redis for reposCacheTTL (5 minutes) to avoid hammering
// the GitHub API on every page load.
func (s *InstallationService) ListRepos(ctx context.Context, installationID int64) ([]sharedgithub.InstallationRepo, error) {
	cacheKey := fmt.Sprintf("%s%d", reposCachePrefix, installationID)

	// ── Cache read ────────────────────────────────────────────────────────
	cached, err := s.rdb.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var repos []sharedgithub.InstallationRepo
		if jsonErr := json.Unmarshal(cached, &repos); jsonErr == nil {
			s.log.Debug("installation repos: cache hit", zap.Int64("installation_id", installationID))
			return repos, nil
		}
	}

	// ── Cache miss: fetch from GitHub ─────────────────────────────────────
	s.log.Debug("installation repos: cache miss — fetching from GitHub", zap.Int64("installation_id", installationID))
	repos, err := s.ghClient.ListInstallationRepos(ctx, installationID)
	if err != nil {
		return nil, fmt.Errorf("installation: list repos: %w", err)
	}

	// ── Store in Redis (best-effort) ──────────────────────────────────────
	if raw, marshalErr := json.Marshal(repos); marshalErr == nil {
		if setErr := s.rdb.Set(ctx, cacheKey, raw, reposCacheTTL).Err(); setErr != nil {
			s.log.Warn("installation repos: failed to cache", zap.Error(setErr))
		}
	}

	return repos, nil
}
