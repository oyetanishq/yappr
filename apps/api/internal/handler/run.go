package handler

import (
	"context"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	runsvc "github.com/oyetanishq/yappr/apps/api/internal/service/run"
	"github.com/oyetanishq/yappr/apps/shared/config"
	"github.com/oyetanishq/yappr/apps/shared/model"
	"github.com/oyetanishq/yappr/apps/shared/pkg/response"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type runHandler struct {
	runSvc *runsvc.RunService
	log    *zap.Logger
}

func newRunHandler(client *mongo.Client, log *zap.Logger, cfg *config.Config) *runHandler {
	return &runHandler{
		runSvc: runsvc.NewRunService(client, cfg, log),
		log:    log,
	}
}

// List  GET /api/v1/runs
//
// Returns the authenticated user's PR review runs, newest first. The large
// review-content fields are omitted; fetch a single run for the full detail.
func (h *runHandler) List(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	user := c.MustGet("user").(*model.User)

	runs, err := h.runSvc.ListByUser(ctx, user.ID)
	if err != nil {
		h.log.Error("runs: list", zap.String("user", user.ID), zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, runs)
}

// Get  GET /api/v1/runs/:id
//
// Returns a single run with its full review content, scoped to the owner.
func (h *runHandler) Get(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	user := c.MustGet("user").(*model.User)
	id := c.Param("id")

	run, err := h.runSvc.GetByUser(ctx, user.ID, id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			response.NotFound(c)
			return
		}
		h.log.Error("runs: get", zap.String("user", user.ID), zap.String("id", id), zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, run)
}
