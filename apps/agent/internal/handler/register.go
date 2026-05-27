package handler

import (
	"github.com/oyetanishq/yappr/apps/shared/config"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Register wires all route groups onto the engine.
func Register(r *gin.Engine, rdb *redis.Client, client *mongo.Client, log *zap.Logger, cfg *config.Config) {
	// Health – no auth required
	r.GET("/health", healthHandler)

	// API v1
	v1 := r.Group("/api/v1")
	{

		// ── GitHub App ────────────────────────────────────────────────────────
		githubH, err := newGithubHandler(rdb, client, log, cfg)
		if err != nil {
			log.Fatal("failed to initialise github handler", zap.Error(err))
		}

		gh := v1.Group("/github")
		{
			// Receive all GitHub App webhook events (PR opened, closed, etc.).
			// No session auth — secured by HMAC-SHA256 signature verification.
			gh.POST("/webhook", githubH.Webhook)
		}
	}
}
