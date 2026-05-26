package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	App       AppConfig
	Redis     RedisConfig
	Mongo     MongoConfig
	GithubApp GithubAppConfig
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
	AppID          string
	ClientID       string
	ClientSecret   string
	PrivateKeyPath string
	WebhookSecret  string
	AppName        string
}

// Load reads .env (if present) then falls back to environment variables.
func Load() (*Config, error) {
	// .env is optional — in Docker the env vars are injected directly
	_ = godotenv.Load()

	origins := strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:5173"), ",")

	return &Config{
		App: AppConfig{
			Port:           getEnv("PORT", "8080"),
			Env:            getEnv("APP_ENV", "development"),
			AllowedOrigins: origins,
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "redis:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		},
		GithubApp: GithubAppConfig{
			AppID:          getEnv("GITHUB_APP_ID", ""),
			ClientID:       getEnv("GITHUB_APP_CLIENT_ID", ""),
			ClientSecret:   getEnv("GITHUB_APP_CLIENT_SECRET", ""),
			PrivateKeyPath: getEnv("GITHUB_APP_PRIVATE_KEY_PATH", ""),
			WebhookSecret:  getEnv("GITHUB_WEBHOOK_SECRET", ""),
			AppName:        getEnv("GITHUB_APP_NAME", ""),
		},
		Mongo: MongoConfig{
			URI: getEnv("MONGODB_URI", "mongodb://mongo:27017"),
			DB:  getEnv("MONGODB_DB", "yappr"),
		},
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
