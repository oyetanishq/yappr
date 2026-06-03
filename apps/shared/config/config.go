package config

import (
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App       AppConfig
	Auth      AuthConfig
	Redis     RedisConfig
	Mongo     MongoConfig
	GithubApp GithubAppConfig
	OpenAI    OpenAIConfig
}

type AppConfig struct {
	Port           string
	Env            string
	AllowedOrigins []string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type MongoConfig struct {
	URI string
	DB  string
}

type GithubAppConfig struct {
	AppID         string
	ClientID      string
	ClientSecret  string
	PrivateKey    string
	WebhookSecret string
	AppName       string
	CallbackURL   string
}

type AuthConfig struct {
	JWTSecret  string
	SessionTTL time.Duration
}

type OpenAIConfig struct {
	APIKey  string
	BaseURL string
}

// Load reads .env (if present) then falls back to environment variables.
// .env is optional — in Docker the env vars are injected directly.
func Load() (*Config, error) {
	_ = godotenv.Load()

	origins := strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:5173"), ",")

	return &Config{
		App: AppConfig{
			Port:           getEnv("PORT", "8080"),
			Env:            getEnv("APP_ENV", "development"),
			AllowedOrigins: origins,
		},
		Auth: AuthConfig{
			JWTSecret:  getEnv("JWT_SECRET", "change-me-in-production"),
			SessionTTL: 7 * 24 * time.Hour,
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "redis:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		},
		GithubApp: GithubAppConfig{
			AppID:         getEnv("GITHUB_APP_ID", ""),
			ClientID:      getEnv("GITHUB_APP_CLIENT_ID", ""),
			ClientSecret:  getEnv("GITHUB_APP_CLIENT_SECRET", ""),
			PrivateKey:    getEnv("GITHUB_APP_PRIVATE_KEY", ""),
			WebhookSecret: getEnv("GITHUB_WEBHOOK_SECRET", ""),
			AppName:       getEnv("GITHUB_APP_NAME", ""),
			CallbackURL:   getEnv("GITHUB_OAUTH_CALLBACK_URL", "http://localhost:8080/api/v1/auth/github/callback"),
		},
		Mongo: MongoConfig{
			URI: getEnv("MONGODB_URI", "mongodb://mongo:27017"),
			DB:  getEnv("MONGODB_DB", "yappr"),
		},
		OpenAI: OpenAIConfig{
			APIKey:  getEnv("OPENAI_API_KEY", ""),
			BaseURL: getEnv("OPENAI_BASE_URL", ""),
		},
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
