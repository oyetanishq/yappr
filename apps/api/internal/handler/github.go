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
)

type githubHandler struct {
	installSvc *githubsvc.InstallationService
	rdb        *redis.Client
	cfg        *config.Config
	log        *zap.Logger
	http       *http.Client
}

func newGithubHandler(rdb *redis.Client, client *mongo.Client, log *zap.Logger, cfg *config.Config) (*githubHandler, error) {
	installSvc, err := githubsvc.NewInstallationService(rdb, client, cfg, log)
	if err != nil {
		return nil, fmt.Errorf("github handler: init installation service: %w", err)
	}

	return &githubHandler{
		installSvc: installSvc,
		rdb:        rdb,
		cfg:        cfg,
		log:        log,
		http:       &http.Client{Timeout: 10 * time.Second},
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
	frontendURL := h.cfg.App.FrontendURL

	// -- CSRF check
	state := c.Query("state")
	if err := h.validateInstallState(ctx, state, user.ID); err != nil {
		h.log.Warn("github install callback: invalid state", zap.Error(err))
		c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/dashboard?error=invalid_state")
		return
	}

	// -- Parse installation_id
	rawID := c.Query("installation_id")
	if rawID == "" {
		// User cancelled the install flow on GitHub's side.
		c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/dashboard")
		return
	}

	installationID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/dashboard?error=invalid_installation_id")
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
	_, err = h.installSvc.Save(ctx, installationID, user.ID, accountLogin)
	if err != nil {
		h.log.Error("github install callback: save installation", zap.Error(err))
		c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/dashboard?error=internal_error")
		return
	}

	h.log.Info("github app installed",
		zap.Int64("installation_id", installationID),
		zap.String("user_id", user.ID),
		zap.String("account", accountLogin),
	)

	c.Redirect(http.StatusTemporaryRedirect, frontendURL+"/dashboard")
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

// InstallationRepos  GET /api/v1/github/installations/:id/repos
//
// Returns all repositories accessible to a specific GitHub App installation.
// Ownership is verified: the installation must belong to the authenticated user.
// Results are served from a 5-minute Redis cache to avoid GitHub rate-limits.
func (h *githubHandler) InstallationRepos(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	user := c.MustGet("user").(*model.User)

	rawID := c.Param("id")
	installationID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid installation id")
		return
	}

	// Ownership check — the installation must belong to this user.
	inst, err := h.installSvc.GetByInstallationID(ctx, installationID)
	if err != nil {
		h.log.Warn("installation repos: not found", zap.Int64("installation_id", installationID), zap.Error(err))
		response.NotFound(c)
		return
	}
	if inst.UserID != user.ID {
		response.Forbidden(c)
		return
	}

	repos, err := h.installSvc.ListRepos(ctx, installationID)
	if err != nil {
		h.log.Error("installation repos: list", zap.Int64("installation_id", installationID), zap.Error(err))
		response.InternalError(c)
		return
	}
	response.OK(c, repos)
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

	resp, err := h.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch installation account: http: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch installation account: github returned %d", resp.StatusCode)
	}

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
