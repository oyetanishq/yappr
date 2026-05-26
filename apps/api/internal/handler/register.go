package handler

import (
	"github.com/oyetanishq/yappr/apps/api/internal/config"
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
		// Example resource group
		exampleH := newExampleHandler(rdb, log)
		ex := v1.Group("/example")
		{
			ex.GET("", exampleH.List)
			ex.POST("", exampleH.Create)
			ex.GET("/:id", exampleH.Get)
		}
	}
}
