package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/oyetanishq/yappr/apps/api/internal/config"
	"github.com/oyetanishq/yappr/apps/api/internal/model"
	authsvc "github.com/oyetanishq/yappr/apps/api/internal/service/auth"
	"github.com/oyetanishq/yappr/apps/api/pkg/response"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

const (
	sessionCookie = "__session"
	sessionPrefix = "session:"
)

// sessionClaims are the JWT claims we embed in the session cookie.
type sessionClaims struct {
	jwt.RegisteredClaims
}

type authHandler struct {
	oauth *authsvc.OAuthService
	rdb   *redis.Client
	cfg   *config.Config
	log   *zap.Logger
}

func newAuthHandler(rdb *redis.Client, client *mongo.Client, log *zap.Logger, cfg *config.Config) (*authHandler, error) {
	svc, err := authsvc.New(rdb, client, cfg, log)
	if err != nil {
		return nil, err
	}
	return &authHandler{oauth: svc, rdb: rdb, cfg: cfg, log: log}, nil
}

// Redirect  GET /api/v1/auth/github
// Generates a CSRF state token and redirects the browser to GitHub's OAuth
// authorization page.
func (h *authHandler) Redirect(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	state, err := h.oauth.GenerateState(ctx)
	if err != nil {
		h.log.Error("oauth generate state", zap.Error(err))
		response.InternalError(c)
		return
	}

	params := url.Values{}
	params.Set("client_id", h.cfg.GithubApp.ClientID)
	params.Set("redirect_uri", h.cfg.GithubApp.CallbackURL)
	params.Set("scope", "read:user user:email")
	params.Set("state", state)

	location := "https://github.com/login/oauth/authorize?" + params.Encode()
	c.Redirect(http.StatusTemporaryRedirect, location)
}

// Callback  GET /api/v1/auth/github/callback
// Validates the CSRF state, exchanges the code for a GitHub access token,
// fetches the GitHub user, upserts the user in MongoDB, and sets a session cookie.
func (h *authHandler) Callback(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	// -- CSRF check
	state := c.Query("state")
	if err := h.oauth.ValidateState(ctx, state); err != nil {
		h.log.Warn("oauth invalid state", zap.Error(err))
		response.BadRequest(c, "invalid or expired oauth state")
		return
	}

	code := c.Query("code")
	if code == "" {
		response.BadRequest(c, "missing oauth code")
		return
	}

	// -- Exchange code for access token
	accessToken, err := h.oauth.ExchangeCode(ctx, code)
	if err != nil {
		h.log.Error("oauth exchange code", zap.Error(err))
		response.InternalError(c)
		return
	}

	// -- Fetch GitHub user
	ghUser, err := h.oauth.GetGithubUser(ctx, accessToken)
	if err != nil {
		h.log.Error("oauth get github user", zap.Error(err))
		response.InternalError(c)
		return
	}

	// -- Upsert into MongoDB
	user, err := h.oauth.UpsertUser(ctx, ghUser)
	if err != nil {
		h.log.Error("oauth upsert user", zap.Error(err))
		response.InternalError(c)
		return
	}

	// -- Create session
	if err := h.createSession(c, user); err != nil {
		h.log.Error("oauth create session", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, user)
}

// Me  GET /api/v1/auth/me
// Returns the currently authenticated user.
// The RequireAuth middleware must have already set "user" on the context.
func (h *authHandler) Me(c *gin.Context) {
	user, _ := c.Get("user")
	response.OK(c, user)
}

// Logout  POST /api/v1/auth/logout
// Deletes the server-side session from Redis and clears the cookie.
func (h *authHandler) Logout(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Best-effort: delete the session from Redis if we can extract the jti.
	if token, err := c.Cookie(sessionCookie); err == nil {
		if claims, err := h.parseJWT(token); err == nil {
			jti := claims.ID
			if jti != "" {
				_ = h.rdb.Del(ctx, sessionKey(jti)).Err()
			}
		}
	}

	h.clearSessionCookie(c)
	response.OK(c, gin.H{"message": "logged out"})
}

// ── session helpers ───────────────────────────────────────────────────────────

// createSession mints a JWT (jti = new UUID), stores the serialised user in
// Redis (TTL = SessionTTL), and sets the HttpOnly session cookie.
func (h *authHandler) createSession(c *gin.Context, user *model.User) error {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	jti := uuid.NewString()
	exp := time.Now().Add(h.cfg.Auth.SessionTTL)

	claims := sessionClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "yappr",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(h.cfg.Auth.JWTSecret))
	if err != nil {
		return fmt.Errorf("sign jwt: %w", err)
	}

	// Serialize user and store in Redis for server-side revocation + fast lookup.
	userJSON, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("marshal user: %w", err)
	}
	if err := h.rdb.Set(ctx, sessionKey(jti), userJSON, h.cfg.Auth.SessionTTL).Err(); err != nil {
		return fmt.Errorf("store session: %w", err)
	}

	secure := h.cfg.App.Env == "production"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookie, signed, int(h.cfg.Auth.SessionTTL.Seconds()), "/", "", secure, true)
	return nil
}

// parseJWT validates the token signature and returns the claims.
func (h *authHandler) parseJWT(raw string) (*sessionClaims, error) {
	token, err := jwt.ParseWithClaims(raw, &sessionClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(h.cfg.Auth.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*sessionClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}

// clearSessionCookie expires the cookie immediately.
func (h *authHandler) clearSessionCookie(c *gin.Context) {
	secure := h.cfg.App.Env == "production"
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(sessionCookie, "", -1, "/", "", secure, true)
}

// sessionKey returns the Redis key for a given session ID.
func sessionKey(jti string) string { return sessionPrefix + jti }
