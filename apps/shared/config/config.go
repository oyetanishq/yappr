package config

import (
	"fmt"
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
	LLM       LLMConfig
	Razorpay  RazorpayConfig
}

type AppConfig struct {
	Port           string
	Env            string
	AllowedOrigins []string
	FrontendURL    string
	// AgentURL is the base URL of the agent service, used by the API's /health
	// endpoint to probe agent liveness. In Docker this is http://agent:8081.
	AgentURL string
}

type RedisConfig struct {
	URL string
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

// LLMConfig configures the chat-completion provider. The client speaks the
// OpenAI wire format, so any OpenAI-compatible endpoint works (Gemini, etc.).
// BugModel optionally overrides BaseModel for the deep bug-detection pass.
type LLMConfig struct {
	APIKey    string
	BaseURL   string
	BaseModel string
	BugModel  string
}

type RazorpayConfig struct {
	KeyID         string
	KeySecret     string
	PlanID        string
	WebhookSecret string
}

// Load reads .env (if present) then falls back to environment variables.
// .env is optional — in Docker the env vars are injected directly.
func Load(envFiles ...string) (*Config, error) {
	_ = godotenv.Load(envFiles...)

	origins := strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:5173"), ",")

	return &Config{
		App: AppConfig{
			Port:           getEnv("PORT", "8080"),
			Env:            getEnv("APP_ENV", "development"),
			AllowedOrigins: origins,
			FrontendURL:    getEnv("FRONTEND_URL", "http://localhost:5173"),
			AgentURL:       getEnv("AGENT_URL", "http://localhost:8081"),
		},
		Auth: AuthConfig{
			JWTSecret:  getEnv("JWT_SECRET", "change-me-in-production"),
			SessionTTL: 7 * 24 * time.Hour,
		},
		Redis: RedisConfig{
			URL: getEnv("REDIS_URL", "redis://localhost:6379"),
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
			URI: getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			DB:  getEnv("MONGODB_DB", "yappr"),
		},
		LLM: LLMConfig{
			APIKey:    getEnv("LLM_API_KEY", ""),
			BaseURL:   getEnv("LLM_BASE_URL", ""),
			BaseModel: getEnv("LLM_BASE_MODEL", ""),
			BugModel:  getEnv("LLM_BUG_MODEL", ""),
		},
		Razorpay: RazorpayConfig{
			KeyID:         getEnv("RAZORPAY_KEY_ID", ""),
			KeySecret:     getEnv("RAZORPAY_KEY_SECRET", ""),
			PlanID:        getEnv("RAZORPAY_PLAN_ID", ""),
			WebhookSecret: getEnv("RAZORPAY_WEBHOOK_SECRET", ""),
		},
	}, nil
}

// Validate checks that security-critical configuration is present. In production
// it fails fast on empty/default secrets so the service never boots with a
// forgeable webhook signature or a default JWT secret. Local dev stays permissive.
func (c *Config) Validate() error {
	if c.App.Env != "production" {
		return nil
	}

	var missing []string
	require := func(name, value string) {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}

	require("RAZORPAY_KEY_ID", c.Razorpay.KeyID)
	require("RAZORPAY_KEY_SECRET", c.Razorpay.KeySecret)
	require("RAZORPAY_PLAN_ID", c.Razorpay.PlanID)
	require("RAZORPAY_WEBHOOK_SECRET", c.Razorpay.WebhookSecret)

	if strings.TrimSpace(c.Auth.JWTSecret) == "" || c.Auth.JWTSecret == "change-me-in-production" {
		missing = append(missing, "JWT_SECRET")
	}

	if len(missing) > 0 {
		return fmt.Errorf("config: missing/insecure required settings in production: %s", strings.Join(missing, ", "))
	}
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
