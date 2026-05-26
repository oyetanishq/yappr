package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/oyetanishq/yappr/apps/api/internal/config"
	"github.com/oyetanishq/yappr/apps/api/internal/model"
	"github.com/oyetanishq/yappr/apps/api/pkg/response"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	authCookieName = "__session"
	authSessionPfx = "session:"
)

type authClaims struct {
	jwt.RegisteredClaims
}

// RequireAuth validates the session cookie, fetches the user JSON stored in
// Redis, and sets "user" on the gin context. No MongoDB round-trip is made;
// the session Redis key holds the serialised model.User written at login.
func RequireAuth(rdb *redis.Client, log *zap.Logger, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := c.Cookie(authCookieName)
		if err != nil {
			response.Unauthorized(c)
			return
		}

		// -- Parse + verify JWT
		claims, err := parseToken(raw, cfg.Auth.JWTSecret)
		if err != nil {
			log.Debug("auth: invalid jwt", zap.Error(err))
			response.Unauthorized(c)
			return
		}

		jti := claims.ID
		if jti == "" {
			response.Unauthorized(c)
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		// -- Fetch user JSON from Redis (GET returns redis.Nil if revoked/expired)
		userJSON, err := rdb.Get(ctx, authSessionPfx+jti).Bytes()
		if err != nil {
			// redis.Nil means session was revoked or expired.
			log.Debug("auth: session not found", zap.String("jti", jti), zap.Error(err))
			response.Unauthorized(c)
			return
		}

		var user model.User
		if err := json.Unmarshal(userJSON, &user); err != nil {
			log.Error("auth: unmarshal user", zap.Error(err))
			response.InternalError(c)
			return
		}

		c.Set("user", &user)
		c.Next()
	}
}

func parseToken(raw, secret string) (*authClaims, error) {
	token, err := jwt.ParseWithClaims(raw, &authClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*authClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid claims")
	}
	return claims, nil
}
