package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/oyetanishq/yappr/apps/api/internal/config"
	"github.com/oyetanishq/yappr/apps/api/pkg/response"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

const (
	authCookieName = "__session"
	authSessionPfx = "session:"
)

type authClaims struct {
	jwt.RegisteredClaims
}

// RequireAuth validates the session cookie, checks the Redis session store,
// and loads the user from MongoDB. Sets "user" on the gin context on success.
func RequireAuth(rdb *redis.Client, client *mongo.Client, log *zap.Logger, cfg *config.Config) gin.HandlerFunc {
	col := client.Database(cfg.Mongo.DB).Collection("users")

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
		userID := claims.Subject

		if jti == "" || userID == "" {
			response.Unauthorized(c)
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		// -- Check Redis session (server-side revocation)
		exists, err := rdb.Exists(ctx, authSessionPfx+jti).Result()
		if err != nil {
			log.Error("auth: redis session check", zap.Error(err))
			response.InternalError(c)
			return
		}
		if exists == 0 {
			// Session was revoked (e.g. logout).
			response.Unauthorized(c)
			return
		}

		// -- Load user from MongoDB
		var user bson.M
		if err := col.FindOne(ctx, bson.D{{Key: "_id", Value: userID}}).Decode(&user); err != nil {
			if err == mongo.ErrNoDocuments {
				response.Unauthorized(c)
				return
			}
			log.Error("auth: load user", zap.Error(err))
			response.InternalError(c)
			return
		}

		c.Set("user", user)
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
