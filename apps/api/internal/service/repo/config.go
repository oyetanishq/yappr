// Package repo provides a service for managing per-repository Yappr configuration.
package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/oyetanishq/yappr/apps/shared/config"
	"github.com/oyetanishq/yappr/apps/shared/model"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const (
	repoConfigCachePrefix = "repo_config:"
	repoConfigCacheTTL    = 10 * time.Minute
)

// ConfigService manages per-repository configuration in MongoDB,
// with a Redis read-through cache to avoid redundant DB hits.
type ConfigService struct {
	col *mongo.Collection
	rdb *redis.Client
	log *zap.Logger
}

// NewConfigService creates the service and ensures the required MongoDB index exists.
func NewConfigService(rdb *redis.Client, client *mongo.Client, cfg *config.Config, log *zap.Logger) (*ConfigService, error) {
	col := client.Database(cfg.Mongo.DB).Collection("repo_configs")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Unique index on repo_full_name so each repo has exactly one config document.
	if _, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "repo_full_name", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("repo_full_name_unique"),
	}); err != nil {
		log.Warn("mongo repo_configs index: repo_full_name", zap.Error(err))
	}

	return &ConfigService{col: col, rdb: rdb, log: log}, nil
}

// Get returns the config for a repo. If none exists, it returns a default config (not persisted).
// Results are served from Redis cache (10-min TTL) to avoid redundant Mongo reads.
func (s *ConfigService) Get(ctx context.Context, repoFullName string) (*model.RepoConfig, error) {
	cacheKey := repoConfigCachePrefix + repoFullName

	// ── Cache read ──────────────────────────────────────────────────────────
	if cached, err := s.rdb.Get(ctx, cacheKey).Bytes(); err == nil {
		var cfg model.RepoConfig
		if jsonErr := json.Unmarshal(cached, &cfg); jsonErr == nil {
			s.log.Debug("repo_config: cache hit", zap.String("repo", repoFullName))
			return &cfg, nil
		}
	}

	// ── Mongo read ──────────────────────────────────────────────────────────
	var cfg model.RepoConfig
	err := s.col.FindOne(ctx, bson.D{{Key: "repo_full_name", Value: repoFullName}}).Decode(&cfg)
	if err == mongo.ErrNoDocuments {
		// Return a default config without persisting it.
		return defaultConfig(repoFullName), nil
	}
	if err != nil {
		return nil, fmt.Errorf("repo_config: get %q: %w", repoFullName, err)
	}

	// ── Backfill cache ──────────────────────────────────────────────────────
	s.cacheConfig(ctx, cacheKey, &cfg)

	return &cfg, nil
}

// Upsert creates or updates the config for the given repo, then invalidates the Redis cache.
func (s *ConfigService) Upsert(ctx context.Context, userID, repoFullName string, ignoredPaths []string, personality model.Personality) (*model.RepoConfig, error) {
	if !personality.IsValid() {
		personality = model.DefaultPersonality
	}
	if ignoredPaths == nil {
		ignoredPaths = []string{}
	}

	now := time.Now().UTC()
	filter := bson.D{{Key: "repo_full_name", Value: repoFullName}}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "user_id", Value: userID},
			{Key: "ignored_paths", Value: ignoredPaths},
			{Key: "personality", Value: personality},
			{Key: "updated_at", Value: now},
		}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "_id", Value: uuid.NewString()},
			{Key: "repo_full_name", Value: repoFullName},
			{Key: "created_at", Value: now},
		}},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var result model.RepoConfig
	if err := s.col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&result); err != nil {
		return nil, fmt.Errorf("repo_config: upsert %q: %w", repoFullName, err)
	}

	// Invalidate stale cache entry so the next read gets fresh data from Mongo.
	cacheKey := repoConfigCachePrefix + repoFullName
	if err := s.rdb.Del(ctx, cacheKey).Err(); err != nil {
		s.log.Warn("repo_config: failed to invalidate cache", zap.String("repo", repoFullName), zap.Error(err))
	}

	return &result, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func defaultConfig(repoFullName string) *model.RepoConfig {
	return &model.RepoConfig{
		RepoFullName: repoFullName,
		IgnoredPaths: []string{},
		Personality:  model.DefaultPersonality,
	}
}

func (s *ConfigService) cacheConfig(ctx context.Context, key string, cfg *model.RepoConfig) {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return
	}
	if err := s.rdb.Set(ctx, key, raw, repoConfigCacheTTL).Err(); err != nil {
		s.log.Warn("repo_config: failed to cache", zap.String("key", key), zap.Error(err))
	}
}
