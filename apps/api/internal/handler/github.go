package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	githubsvc "github.com/oyetanishq/yappr/apps/api/internal/service/github"
	"github.com/oyetanishq/yappr/apps/shared/config"
	"github.com/oyetanishq/yappr/apps/shared/model"
	"github.com/oyetanishq/yappr/apps/shared/pkg/response"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

const (
	installStatePrefix = "github:install:state:"
	installStateTTL    = 10 * time.Minute
	// maxWebhookBody limits how many bytes we read from a webhook POST to
	// prevent memory exhaustion from oversized payloads.
	maxWebhookBody = 25 << 20 // 25 MB
)

type githubHandler struct {
	installSvc *githubsvc.InstallationService
	rdb        *redis.Client
	cfg        *config.Config
	log        *zap.Logger
}

func newGithubHandler(rdb *redis.Client, client *mongo.Client, log *zap.Logger, cfg *config.Config) (*githubHandler, error) {
	installSvc, err := githubsvc.NewInstallationService(client, cfg, log)
	if err != nil {
		return nil, fmt.Errorf("github handler: init installation service: %w", err)
	}

	return &githubHandler{
		installSvc: installSvc,
		rdb:        rdb,
		cfg:        cfg,
		log:        log,
	}, nil
}

// Install  GET /api/v1/github/install
//
// Generates a CSRF state token, stores it in Redis, and redirects the
// authenticated user to GitHub's native repo-selection install page.
// GitHub hosts the entire repo-picker UI — the user selects which repositories
// to grant your app access to, then GitHub redirects back to InstallCallback.
func (h *githubHandler) Install(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	user := c.MustGet("user").(*model.User)

	// Mint a CSRF state token bound to this user's ID.
	state, err := h.generateInstallState(ctx, user.ID)
	if err != nil {
		h.log.Error("github install: generate state", zap.Error(err))
		response.InternalError(c)
		return
	}

	params := url.Values{}
	params.Set("state", state)

	// GitHub's native repo-picker for app installations.
	location := fmt.Sprintf(
		"https://github.com/apps/%s/installations/new?%s",
		h.cfg.GithubApp.AppName,
		params.Encode(),
	)
	c.Redirect(http.StatusTemporaryRedirect, location)
}

// InstallCallback  GET /api/v1/github/install/callback
//
// GitHub redirects here after the user completes (or cancels) the repo-picker.
// Query params provided by GitHub:
//
//	installation_id – the new GitHub App installation ID
//	state           – the CSRF token we set in Install
//
// We validate the state token, then persist the installation to MongoDB.
func (h *githubHandler) InstallCallback(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	user := c.MustGet("user").(*model.User)

	// -- CSRF check
	state := c.Query("state")
	if err := h.validateInstallState(ctx, state, user.ID); err != nil {
		h.log.Warn("github install callback: invalid state", zap.Error(err))
		response.BadRequest(c, "invalid or expired install state")
		return
	}

	// -- Parse installation_id
	rawID := c.Query("installation_id")
	if rawID == "" {
		// User cancelled the install flow on GitHub's side.
		response.BadRequest(c, "installation cancelled or installation_id missing")
		return
	}

	installationID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid installation_id")
		return
	}

	// -- Fetch the installation account login from GitHub (best-effort).
	accountLogin, err := h.fetchInstallationAccount(ctx, installationID)
	if err != nil {
		h.log.Warn("github install callback: could not fetch account login",
			zap.Error(err),
			zap.Int64("installation_id", installationID),
		)
		accountLogin = "" // non-fatal — we still save the installation
	}

	// -- Persist to MongoDB
	inst, err := h.installSvc.Save(ctx, installationID, user.ID, accountLogin)
	if err != nil {
		h.log.Error("github install callback: save installation", zap.Error(err))
		response.InternalError(c)
		return
	}

	h.log.Info("github app installed",
		zap.Int64("installation_id", installationID),
		zap.String("user_id", user.ID),
		zap.String("account", accountLogin),
	)

	response.OK(c, inst)
}

// Installations  GET /api/v1/github/installations
//
// Returns all GitHub App installations for the authenticated user.
func (h *githubHandler) Installations(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	user := c.MustGet("user").(*model.User)

	installs, err := h.installSvc.ListForUser(ctx, user.ID)
	if err != nil {
		h.log.Error("github installations: list", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.OK(c, installs)
}

// ── install-state helpers ─────────────────────────────────────────────────────

// generateInstallState mints a UUID CSRF token, stores it in Redis keyed to
// the userID (so a stolen token from user A cannot be used by user B), and
// returns the token.
func (h *githubHandler) generateInstallState(ctx context.Context, userID string) (string, error) {
	token := uuid.NewString()
	key := installStateKey(token)
	if err := h.rdb.Set(ctx, key, userID, installStateTTL).Err(); err != nil {
		return "", fmt.Errorf("github: store install state: %w", err)
	}
	return token, nil
}

// validateInstallState verifies the token exists in Redis and was issued for
// the given userID, then atomically deletes it (one-time use).
func (h *githubHandler) validateInstallState(ctx context.Context, token, userID string) error {
	if token == "" {
		return fmt.Errorf("github: empty install state token")
	}
	key := installStateKey(token)
	storedUserID, err := h.rdb.GetDel(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("github: validate install state: %w", err)
	}
	if storedUserID != userID {
		return fmt.Errorf("github: install state user mismatch")
	}
	return nil
}

func installStateKey(token string) string { return installStatePrefix + token }

// fetchInstallationAccount calls the public GitHub API to get the account login
// associated with an installation_id.
//
// NOTE: For private orgs or to avoid rate-limits, replace this with an
// app-level JWT-authenticated request using the app's private key.
func (h *githubHandler) fetchInstallationAccount(ctx context.Context, installationID int64) (string, error) {
	apiURL := fmt.Sprintf("https://api.github.com/app/installations/%d", installationID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("fetch installation account: build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch installation account: http: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Account struct {
			Login string `json:"login"`
		} `json:"account"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("fetch installation account: decode: %w", err)
	}
	return result.Account.Login, nil
}
