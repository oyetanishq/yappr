package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	githubsvc "github.com/oyetanishq/yappr/apps/agent/internal/service/github"
	reposvc "github.com/oyetanishq/yappr/apps/agent/internal/service/repo"
	"github.com/oyetanishq/yappr/apps/agent/internal/service/reviewer"
	runsvc "github.com/oyetanishq/yappr/apps/agent/internal/service/run"
	usersvc "github.com/oyetanishq/yappr/apps/agent/internal/service/user"
	"github.com/oyetanishq/yappr/apps/shared/config"
	sharedgithub "github.com/oyetanishq/yappr/apps/shared/github"
	"github.com/oyetanishq/yappr/apps/shared/pkg/response"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

const (
	// maxWebhookBody limits how many bytes we read from a webhook POST to
	// prevent memory exhaustion from oversized payloads.
	maxWebhookBody = 25 << 20 // 25 MB
)

type githubHandler struct {
	webhookSvc *githubsvc.WebhookService
	rdb        *redis.Client
	client     *mongo.Client
	cfg        *config.Config
	log        *zap.Logger
}

func newGithubHandler(rdb *redis.Client, client *mongo.Client, log *zap.Logger, cfg *config.Config) (*githubHandler, error) {
	ghClient := sharedgithub.NewClient(cfg.GithubApp.AppID, cfg.GithubApp.PrivateKey)

	// RunService persists PR review run records for the dashboard history.
	runSvc := runsvc.NewRunService(client, cfg, log)

	pipeline := reviewer.NewPipeline(ghClient, runSvc, cfg, log)

	// RepoConfigService gives the webhook access to per-repo config (personality, ignored paths).
	repoConfigSvc := reposvc.NewConfigService(rdb, client, cfg, log)
	userSvc := usersvc.NewUserService(client, cfg, log)
	webhookSvc := githubsvc.NewWebhookService(cfg.GithubApp.WebhookSecret, ghClient, pipeline, repoConfigSvc, userSvc, runSvc, log)

	return &githubHandler{
		webhookSvc: webhookSvc,
		rdb:        rdb,
		client:     client,
		cfg:        cfg,
		log:        log,
	}, nil
}

// Webhook  POST /api/v1/github/webhook
//
// Receives all GitHub App webhook events. We verify the HMAC-SHA256 signature
// using the webhook secret before processing any payload.
// This endpoint requires NO session auth — GitHub calls it directly.
func (h *githubHandler) Webhook(c *gin.Context) {
	// -- Read body with an upper bound to prevent memory exhaustion.
	payload, err := io.ReadAll(io.LimitReader(c.Request.Body, maxWebhookBody))
	if err != nil {
		h.log.Error("webhook: read body", zap.Error(err))
		response.InternalError(c)
		return
	}

	// -- Verify HMAC-SHA256 signature
	sig := c.GetHeader("X-Hub-Signature-256")
	if sig == "" {
		response.BadRequest(c, "missing X-Hub-Signature-256 header")
		return
	}
	if err := h.webhookSvc.VerifySignature(payload, sig); err != nil {
		h.log.Warn("webhook: signature verification failed", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	eventType := c.GetHeader("X-GitHub-Event")
	if eventType == "" {
		response.BadRequest(c, "missing X-GitHub-Event header")
		return
	}

	// -- Dispatch synchronously for routing; review pipeline runs in goroutine.
	if err := h.webhookSvc.Dispatch(c.Request.Context(), eventType, payload); err != nil {
		h.log.Error("webhook: dispatch error",
			zap.String("event", eventType),
			zap.Error(err),
		)
		// Return 200 even on processing errors to prevent GitHub from retrying.
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
