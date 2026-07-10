package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/oyetanishq/yappr/apps/shared/config"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

// serviceHealth is the per-dependency status reported by the /health endpoint.
type serviceHealth struct {
	Status    string `json:"status"` // "ok" | "down"
	LatencyMs int64  `json:"latency_ms"`
	Error     string `json:"error,omitempty"`
}

const healthCheckTimeout = 3 * time.Second

// healthHandler probes Redis, Mongo, and the agent service and reports each
// dependency's status. The overall status is "ok" only when every dependency is
// reachable, otherwise "degraded".
//
// It always responds 200 so container/orchestrator health-checks keep treating
// the API process as live (a downstream outage shouldn't restart the API);
// callers read the per-service fields to render the status page.
func healthHandler(rdb *redis.Client, mongoClient *mongo.Client, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		services := gin.H{
			"redis": checkRedis(c.Request.Context(), rdb),
			"mongo": checkMongo(c.Request.Context(), mongoClient),
			"agent": checkAgent(c.Request.Context(), cfg.App.AgentURL),
		}

		overall := "ok"
		for _, v := range services {
			if v.(serviceHealth).Status != "ok" {
				overall = "degraded"
				break
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   overall,
			"services": services,
		})
	}
}

func checkRedis(ctx context.Context, rdb *redis.Client) serviceHealth {
	ctx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	start := time.Now()
	err := rdb.Ping(ctx).Err()
	return result(start, err)
}

func checkMongo(ctx context.Context, client *mongo.Client) serviceHealth {
	ctx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	start := time.Now()
	err := client.Ping(ctx, nil)
	return result(start, err)
}

func checkAgent(ctx context.Context, baseURL string) serviceHealth {
	ctx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/health", nil)
	if err != nil {
		return result(start, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return result(start, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return serviceHealth{Status: "down", LatencyMs: time.Since(start).Milliseconds(), Error: resp.Status}
	}
	return result(start, nil)
}

func result(start time.Time, err error) serviceHealth {
	sh := serviceHealth{LatencyMs: time.Since(start).Milliseconds()}
	if err != nil {
		sh.Status = "down"
		sh.Error = err.Error()
	} else {
		sh.Status = "ok"
	}
	return sh
}
