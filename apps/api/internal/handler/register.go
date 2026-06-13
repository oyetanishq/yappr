package handler

import (
	"github.com/oyetanishq/yappr/apps/api/internal/middleware"
	"github.com/oyetanishq/yappr/apps/shared/config"
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

		requireAuth := middleware.RequireAuth(rdb, log, cfg)

		auth := v1.Group("/auth")
		{
			auth.GET("/github", authH.Redirect)
			auth.GET("/github/callback", authH.Callback)
			auth.GET("/me", requireAuth, authH.Me)
			auth.POST("/logout", requireAuth, authH.Logout)
			auth.GET("/sessions", requireAuth, authH.Sessions)
			auth.DELETE("/sessions/:id", requireAuth, authH.RevokeSession)
		}

		// ── GitHub App ────────────────────────────────────────────────────────
		githubH, err := newGithubHandler(rdb, client, log, cfg)
		if err != nil {
			log.Fatal("failed to initialise github handler", zap.Error(err))
		}

		gh := v1.Group("/github")
		{
			gh.GET("/install", requireAuth, githubH.Install)
			gh.GET("/install/callback", requireAuth, githubH.InstallCallback)
			gh.GET("/installations", requireAuth, githubH.Installations)
			gh.GET("/installations/:id/repos", requireAuth, githubH.InstallationRepos)
		}

		// ── Repo configuration ────────────────────────────────────────────────
		repoH, err := newRepoHandler(rdb, client, log, cfg)
		if err != nil {
			log.Fatal("failed to initialise repo handler", zap.Error(err))
		}

		repos := v1.Group("/repos")
		{
			repos.GET("/:owner/:repo/config", requireAuth, repoH.GetConfig)
			repos.PUT("/:owner/:repo/config", requireAuth, repoH.UpdateConfig)
		}

		// ── Billing ───────────────────────────────────────────────────────────
		billingH := newBillingHandler(rdb, client, log, cfg)
		billing := v1.Group("/billing")
		{
			// Webhook must be unauthenticated (Razorpay calls it directly).
			billing.POST("/webhook", billingH.Webhook)

			// Subscription management requires a logged-in user.
			billing.POST("/subscribe", requireAuth, billingH.Subscribe)
			billing.POST("/cancel", requireAuth, billingH.Cancel)
		}
	}
}
