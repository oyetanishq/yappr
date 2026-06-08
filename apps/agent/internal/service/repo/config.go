// Package repo provides a read-only view of per-repository Yappr configuration
// for the agent. It reads from MongoDB with a Redis read-through cache.
package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/oyetanishq/yappr/apps/shared/config"
	"github.com/oyetanishq/yappr/apps/shared/model"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

const (
	repoConfigCachePrefix = "repo_config:"
	repoConfigCacheTTL    = 10 * time.Minute
)

// ConfigService provides read-only access to repo configs for the agent.
type ConfigService struct {
	col *mongo.Collection
	rdb *redis.Client
	log *zap.Logger
}

// NewConfigService creates the service. The agent only reads — index creation
// is handled by the API service on startup.
func NewConfigService(rdb *redis.Client, client *mongo.Client, cfg *config.Config, log *zap.Logger) *ConfigService {
	col := client.Database(cfg.Mongo.DB).Collection("repo_configs")
	return &ConfigService{col: col, rdb: rdb, log: log}
}

// Get returns the config for a repo. If none exists, it returns a default config.
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
		return defaultConfig(repoFullName), nil
	}
	if err != nil {
		return nil, fmt.Errorf("repo_config: get %q: %w", repoFullName, err)
	}

	// ── Backfill cache ──────────────────────────────────────────────────────
	if raw, marshalErr := json.Marshal(cfg); marshalErr == nil {
		if setErr := s.rdb.Set(ctx, cacheKey, raw, repoConfigCacheTTL).Err(); setErr != nil {
			s.log.Warn("repo_config: failed to cache", zap.String("key", cacheKey), zap.Error(setErr))
		}
	}

	return &cfg, nil
}

func defaultConfig(repoFullName string) *model.RepoConfig {
	return &model.RepoConfig{
		RepoFullName: repoFullName,
		IgnoredPaths: []string{},
		Personality:  model.DefaultPersonality,
	}
}
