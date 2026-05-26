package handler

import (
	"github.com/oyetanishq/yappr/apps/api/internal/config"
	"github.com/oyetanishq/yappr/apps/api/internal/middleware"
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
		// ── Auth ─────────────────────────────────────────────────────────────
		authH, err := newAuthHandler(rdb, client, log, cfg)
		if err != nil {
			log.Fatal("failed to initialise auth handler", zap.Error(err))
		}

		requireAuth := middleware.RequireAuth(rdb, client, log, cfg)

		auth := v1.Group("/auth")
		{
			auth.GET("/github", authH.Redirect)
			auth.GET("/github/callback", authH.Callback)
			auth.GET("/me", requireAuth, authH.Me)
			auth.POST("/logout", requireAuth, authH.Logout)
		}

		// ── Example resource ──────────────────────────────────────────────────
		exampleH := newExampleHandler(rdb, log)
		ex := v1.Group("/example")
		{
			ex.GET("", exampleH.List)
			ex.POST("", exampleH.Create)
			ex.GET("/:id", exampleH.Get)
		}
	}
}
