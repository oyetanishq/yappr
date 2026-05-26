package db

import (
	"context"
	"fmt"
	"time"

	"github.com/oyetanishq/yappr/apps/api/internal/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// NewMongo creates, connects, and pings a MongoDB client.
func NewMongo(cfg config.MongoConfig) (*mongo.Client, error) {
	// Configure client options
	clientOptions := options.Client().
		ApplyURI(cfg.URI).
		SetConnectTimeout(5 * time.Second).
		SetTimeout(5 * time.Second)

	// Context for the initial connection and ping
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	// Ping the primary node to verify the connection is established
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		// Attempt to disconnect if ping fails to avoid resource leaks
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	return client, nil
}
