// Package auth provides the GitHub OAuth 2.0 service.
//
// Flow:
//  1. GenerateState  → stores a UUID in Redis (TTL 10 min) as CSRF protection
//  2. ValidateState  → fetches + deletes the Redis key (one-time use)
//  3. ExchangeCode   → trades the OAuth code for a GitHub access token
//  4. GetGithubUser  → fetches the authenticated user from the GitHub API
//  5. UpsertUser     → find-or-create the user document in MongoDB
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oyetanishq/yappr/apps/api/internal/config"
	"github.com/oyetanishq/yappr/apps/api/internal/model"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const (
	stateTTL    = 10 * time.Minute
	githubToken = "https://github.com/login/oauth/access_token"
	githubUser  = "https://api.github.com/user"
)

// GithubUser is the subset of fields we care about from the GitHub /user API.
type GithubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// OAuthService handles state management, token exchange, and user persistence.
type OAuthService struct {
	rdb  *redis.Client
	col  *mongo.Collection
	cfg  *config.Config
	log  *zap.Logger
	http *http.Client
}

// New creates a new OAuthService and ensures the github_id unique index exists.
func New(rdb *redis.Client, mongoClient *mongo.Client, cfg *config.Config, log *zap.Logger) (*OAuthService, error) {
	col := mongoClient.Database(cfg.Mongo.DB).Collection("users")

	// Create unique index on github_id once at startup.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	idxModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "github_id", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("github_id_unique"),
	}
	if _, err := col.Indexes().CreateOne(ctx, idxModel); err != nil {
		// Ignore "already exists" — just log it.
		log.Warn("mongo index creation", zap.Error(err))
	}

	return &OAuthService{
		rdb:  rdb,
		col:  col,
		cfg:  cfg,
		log:  log,
		http: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

// GenerateState mints a UUID, stores it in Redis, and returns it.
// The token is single-use and expires in 10 minutes.
func (s *OAuthService) GenerateState(ctx context.Context) (string, error) {
	token := uuid.NewString()
	key := stateKey(token)
	if err := s.rdb.Set(ctx, key, "1", stateTTL).Err(); err != nil {
		return "", fmt.Errorf("oauth: store state: %w", err)
	}
	return token, nil
}

// ValidateState checks Redis for the given token, then deletes it (one-time use).
// Returns an error if the token is missing or expired.
func (s *OAuthService) ValidateState(ctx context.Context, token string) error {
	key := stateKey(token)
	n, err := s.rdb.Del(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("oauth: validate state: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("oauth: invalid or expired state token")
	}
	return nil
}

// ExchangeCode trades the GitHub OAuth code for an access token.
func (s *OAuthService) ExchangeCode(ctx context.Context, code string) (string, error) {
	body := url.Values{}
	body.Set("client_id", s.cfg.GithubApp.ClientID)
	body.Set("client_secret", s.cfg.GithubApp.ClientSecret)
	body.Set("code", code)
	body.Set("redirect_uri", s.cfg.GithubApp.CallbackURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubToken, strings.NewReader(body.Encode()))
	if err != nil {
		return "", fmt.Errorf("oauth: build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("oauth: token exchange: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("oauth: parse token response: %w", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("oauth: github error: %s — %s", result.Error, result.ErrorDesc)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("oauth: empty access token")
	}
	return result.AccessToken, nil
}

// GetGithubUser fetches the authenticated user from the GitHub API.
func (s *OAuthService) GetGithubUser(ctx context.Context, accessToken string) (*GithubUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubUser, nil)
	if err != nil {
		return nil, fmt.Errorf("oauth: build user request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("oauth: fetch github user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oauth: github user API returned %d", resp.StatusCode)
	}

	var gu GithubUser
	if err := json.NewDecoder(resp.Body).Decode(&gu); err != nil {
		return nil, fmt.Errorf("oauth: parse github user: %w", err)
	}
	return &gu, nil
}

// UpsertUser inserts a new user or updates an existing one by github_id.
// Returns the full user document.
func (s *OAuthService) UpsertUser(ctx context.Context, gu *GithubUser) (*model.User, error) {
	now := time.Now().UTC()

	filter := bson.D{{Key: "github_id", Value: gu.ID}}
	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "login", Value: gu.Login},
			{Key: "name", Value: gu.Name},
			{Key: "email", Value: gu.Email},
			{Key: "avatar_url", Value: gu.AvatarURL},
			{Key: "updated_at", Value: now},
		}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "_id", Value: uuid.NewString()},
			{Key: "github_id", Value: gu.ID},
			{Key: "created_at", Value: now},
		}},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var user model.User
	if err := s.col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&user); err != nil {
		return nil, fmt.Errorf("oauth: upsert user: %w", err)
	}
	return &user, nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func stateKey(token string) string { return "oauth:state:" + token }
