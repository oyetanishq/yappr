package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/oyetanishq/yappr/apps/api/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type exampleHandler struct {
	rdb *redis.Client
	log *zap.Logger
}

func newExampleHandler(rdb *redis.Client, log *zap.Logger) *exampleHandler {
	return &exampleHandler{rdb: rdb, log: log}
}

type createExampleReq struct {
	Name  string `json:"name"  binding:"required"`
	Value string `json:"value" binding:"required"`
}

// List  GET /api/v1/example
func (h *exampleHandler) List(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	keys, err := h.rdb.Keys(ctx, "example:*").Result()
	if err != nil {
		h.log.Error("redis keys", zap.Error(err))
		response.InternalError(c)
		return
	}

	items := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		val, err := h.rdb.Get(ctx, k).Result()
		if err != nil {
			continue
		}
		var m map[string]any
		if json.Unmarshal([]byte(val), &m) == nil {
			items = append(items, m)
		}
	}
	response.OK(c, items)
}

// Create  POST /api/v1/example
func (h *exampleHandler) Create(c *gin.Context) {
	var req createExampleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	id := uuid.NewString()
	payload, _ := json.Marshal(map[string]any{
		"id":    id,
		"name":  req.Name,
		"value": req.Value,
	})

	key := fmt.Sprintf("example:%s", id)
	if err := h.rdb.Set(ctx, key, payload, 24*time.Hour).Err(); err != nil {
		h.log.Error("redis set", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.Created(c, gin.H{"id": id})
}

// Get  GET /api/v1/example/:id
func (h *exampleHandler) Get(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	id := c.Param("id")
	key := fmt.Sprintf("example:%s", id)
	val, err := h.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		response.NotFound(c)
		return
	}
	if err != nil {
		h.log.Error("redis get", zap.Error(err))
		response.InternalError(c)
		return
	}

	var item map[string]any
	if err := json.Unmarshal([]byte(val), &item); err != nil {
		response.InternalError(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
}
