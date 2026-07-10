package db

import (
	"context"
	"fmt"
	"time"

	"github.com/oyetanishq/yappr/apps/shared/config"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// NewMongo creates, connects, and pings a MongoDB client.
func NewMongo(cfg config.MongoConfig) (*mongo.Client, error) {
	clientOptions := options.Client().
		ApplyURI(cfg.URI).
		SetConnectTimeout(5 * time.Second).
		SetTimeout(5 * time.Second).
		SetMaxPoolSize(300).
		SetMinPoolSize(20)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	return client, nil
}
