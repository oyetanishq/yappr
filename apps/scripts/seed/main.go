package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/oyetanishq/yappr/apps/shared/config"
	"github.com/oyetanishq/yappr/apps/shared/model"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type sessionClaims struct {
	jwt.RegisteredClaims
}

type UserSession struct {
	UserID   string `json:"user_id"`
	GithubID int64  `json:"github_id"`
	Cookie   string `json:"cookie"`
}

func main() {
	var numUsers int
	var envPath string
	var outPath string
	flag.IntVar(&numUsers, "n", 50, "number of users to seed")
	flag.StringVar(&envPath, "env", "../../.env", "path to .env file")
	flag.StringVar(&outPath, "out", "../k6/users.json", "output json path for k6")
	flag.Parse()

	// Load config
	cfg, err := config.Load(envPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx := context.Background()

	// Connect to Mongo
	clientOpts := options.Client().ApplyURI(cfg.Mongo.URI)
	mongoClient, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		log.Fatalf("failed to connect to mongo: %v", err)
	}
	defer func() {
		if err := mongoClient.Disconnect(ctx); err != nil {
			log.Printf("failed to disconnect mongo: %v", err)
		}
	}()
	db := mongoClient.Database(cfg.Mongo.DB)

	// Connect to Redis
	opt, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		log.Fatalf("failed to parse redis url: %v", err)
	}
	rdb := redis.NewClient(opt)
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to ping redis: %v", err)
	}

	var output []UserSession

	// Make sure we have a JWT secret
	if cfg.Auth.JWTSecret == "" {
		log.Fatalf("JWT_SECRET is not set in config")
	}

	for i := 0; i < numUsers; i++ {
		userID := uuid.NewString()
		ghID := int64(1000000 + i)

		now := time.Now().UTC()

		user := &model.User{
			ID:               userID,
			GithubID:         ghID,
			Login:            fmt.Sprintf("testuser%d", i),
			Name:             fmt.Sprintf("Test User %d", i),
			Email:            fmt.Sprintf("test%d@example.com", i),
			CreatedAt:        now,
			UpdatedAt:        now,
			Plan:             model.PlanFree,
			PRCountResetAt:   time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
			PRCountThisMonth: 0,
		}

		// Insert user to mongo
		_, err = db.Collection("users").InsertOne(ctx, user)
		if err != nil {
			log.Fatalf("failed to insert user: %v", err)
		}

		jti := uuid.NewString()
		exp := now.Add(cfg.Auth.SessionTTL)

		session := &model.Session{
			ID:        jti,
			UserID:    userID,
			UserAgent: "k6-stress-test",
			IP:        "127.0.0.1",
			CreatedAt: now,
			ExpiresAt: exp,
		}

		// Insert session to mongo
		_, err = db.Collection("sessions").InsertOne(ctx, session)
		if err != nil {
			log.Fatalf("failed to insert session: %v", err)
		}

		// Insert session to redis
		userJSON, err := json.Marshal(user)
		if err != nil {
			log.Fatalf("failed to marshal user: %v", err)
		}
		err = rdb.Set(ctx, "session:"+jti, userJSON, cfg.Auth.SessionTTL).Err()
		if err != nil {
			log.Fatalf("failed to insert session to redis: %v", err)
		}

		// Generate JWT
		claims := sessionClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   userID,
				ID:        jti,
				ExpiresAt: jwt.NewNumericDate(exp),
				IssuedAt:  jwt.NewNumericDate(now),
				Issuer:    "yappr",
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signed, err := token.SignedString([]byte(cfg.Auth.JWTSecret))
		if err != nil {
			log.Fatalf("failed to sign jwt: %v", err)
		}

		output = append(output, UserSession{
			UserID:   userID,
			GithubID: ghID,
			Cookie:   signed,
		})
	}

	// Write to JSON
	outJSON, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal output json: %v", err)
	}

	dir := filepath.Dir(outPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("failed to create output directory: %v", err)
	}

	err = os.WriteFile(outPath, outJSON, 0644)
	if err != nil {
		log.Fatalf("failed to write output json: %v", err)
	}

	fmt.Printf("Successfully seeded %d users. Saved to %s\n", numUsers, outPath)
}
