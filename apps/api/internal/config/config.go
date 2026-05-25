package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	App   AppConfig
	Redis RedisConfig
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
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", "yappr_redis_secret"),
			DB:       0,
		},
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
