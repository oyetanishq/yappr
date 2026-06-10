package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	reposvc "github.com/oyetanishq/yappr/apps/api/internal/service/repo"
	"github.com/oyetanishq/yappr/apps/shared/config"
	"github.com/oyetanishq/yappr/apps/shared/model"
	"github.com/oyetanishq/yappr/apps/shared/pkg/response"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type repoHandler struct {
	configSvc *reposvc.ConfigService
	log       *zap.Logger
}

func newRepoHandler(rdb *redis.Client, client *mongo.Client, log *zap.Logger, cfg *config.Config) (*repoHandler, error) {
	configSvc, err := reposvc.NewConfigService(rdb, client, cfg, log)
	if err != nil {
		return nil, fmt.Errorf("repo handler: init config service: %w", err)
	}
	return &repoHandler{configSvc: configSvc, log: log}, nil
}

// GetConfig  GET /api/v1/repos/:owner/:repo/config
//
// Returns the current configuration for the specified repository.
// If no config has been saved, returns the default (senior_dev personality, no ignored paths).
func (h *repoHandler) GetConfig(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	repoFullName := c.Param("owner") + "/" + c.Param("repo")

	cfg, err := h.configSvc.Get(ctx, repoFullName)
	if err != nil {
		h.log.Error("repo config: get", zap.String("repo", repoFullName), zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, cfg)
}

// UpdateConfig  PUT /api/v1/repos/:owner/:repo/config
//
// Creates or updates the configuration for the specified repository.
// Accepted body:
//
//	{
//	  "ignored_paths": ["dist/", "node_modules/", "**/*.lock"],
//	  "personality":   "senior_dev"   // bestie | senior_dev | sigma | toxic_tech_lead
//	}
func (h *repoHandler) UpdateConfig(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	user := c.MustGet("user").(*model.User)
	repoFullName := c.Param("owner") + "/" + c.Param("repo")

	var body struct {
		IgnoredPaths []string          `json:"ignored_paths"`
		Personality  model.Personality `json:"personality"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Free-tier users may only use the default (senior_dev) personality.
	if !user.IsPro() && body.Personality != model.DefaultPersonality && body.Personality != "" {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error":    "personality selection requires a Pro subscription",
			"required": "pro",
		})
		return
	}

	cfg, err := h.configSvc.Upsert(ctx, user.ID, repoFullName, body.IgnoredPaths, body.Personality)
	if err != nil {
		h.log.Error("repo config: upsert", zap.String("repo", repoFullName), zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, cfg)
}
