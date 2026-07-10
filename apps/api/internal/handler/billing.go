package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	billingsvc "github.com/oyetanishq/yappr/apps/api/internal/service/billing"
	"github.com/oyetanishq/yappr/apps/shared/config"
	"github.com/oyetanishq/yappr/apps/shared/model"
	"github.com/oyetanishq/yappr/apps/shared/pkg/response"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type billingHandler struct {
	svc *billingsvc.Service
	log *zap.Logger
}

func newBillingHandler(rdb *redis.Client, client *mongo.Client, log *zap.Logger, cfg *config.Config) *billingHandler {
	svc := billingsvc.New(rdb, client, cfg, log)
	return &billingHandler{svc: svc, log: log}
}

// Subscribe  POST /api/v1/billing/subscribe
//
// Creates a Razorpay subscription for the authenticated user and returns
// the hosted checkout URL. The frontend redirects the user to that URL to
// complete payment. The webhook activates the Pro plan once payment succeeds.
func (h *billingHandler) Subscribe(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	user := c.MustGet("user").(*model.User)

	result, err := h.svc.CreateSubscription(ctx, user)
	if err != nil {
		h.log.Error("billing: subscribe", zap.String("user_id", user.ID), zap.Error(err))
		if errors.Is(err, billingsvc.ErrAlreadyPro) {
			response.Conflict(c, "already subscribed to Pro")
			return
		}
		response.InternalError(c)
		return
	}

	response.OK(c, result)
}

// Cancel  POST /api/v1/billing/cancel
//
// Schedules cancellation of the active Razorpay subscription at the end of
// the current billing cycle. The user retains Pro access until then.
func (h *billingHandler) Cancel(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	user := c.MustGet("user").(*model.User)

	if !user.IsPro() {
		response.BadRequest(c, "no active Pro subscription")
		return
	}

	if err := h.svc.CancelSubscription(ctx, user); err != nil {
		h.log.Error("billing: cancel", zap.String("user_id", user.ID), zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"message": "subscription will be cancelled at end of billing cycle"})
}

// Resume  POST /api/v1/billing/resume
//
// Undoes a scheduled cancellation while the user is still within their paid period,
// re-enabling automatic renewal. Only valid when a cancellation is actually pending.
func (h *billingHandler) Resume(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	user := c.MustGet("user").(*model.User)

	if err := h.svc.ReactivateSubscription(ctx, user); err != nil {
		h.log.Error("billing: resume", zap.String("user_id", user.ID), zap.Error(err))
		if errors.Is(err, billingsvc.ErrNotCancelling) {
			response.BadRequest(c, "no scheduled cancellation to resume")
			return
		}
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"message": "subscription resumed"})
}

// webhookEvent is the envelope Razorpay sends for all webhook calls.
type webhookEvent struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

// subscriptionPayload is the nested shape inside Razorpay webhook payloads for subscription events.
type subscriptionPayload struct {
	Subscription struct {
		Entity struct {
			ID    string `json:"id"`
			Notes struct {
				UserID string `json:"user_id"`
			} `json:"notes"`
		} `json:"entity"`
	} `json:"subscription"`
}

// Webhook  POST /api/v1/billing/webhook
//
// Receives Razorpay webhook events. The raw body is verified via HMAC-SHA256
// before processing. This endpoint must NOT be behind the RequireAuth middleware.
func (h *billingHandler) Webhook(c *gin.Context) {
	const maxWebhookBody = 1 << 20 // 1 MB
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, int64(maxWebhookBody)+1))
	if err != nil {
		response.BadRequest(c, "cannot read body")
		return
	}
	if len(body) > maxWebhookBody {
		response.RequestEntityTooLarge(c, "payload too large")
		return
	}

	// ── HMAC verification ─────────────────────────────────────────────────────
	sig := c.GetHeader("X-Razorpay-Signature")
	if !h.svc.VerifyWebhookSignature(body, sig) {
		h.log.Warn("billing: webhook signature mismatch")
		response.Unauthorized(c)
		return
	}

	var event webhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		response.BadRequest(c, "malformed payload")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	h.log.Info("billing: webhook received", zap.String("event", event.Event))

	// On a persistence failure we return a non-2xx status so Razorpay redelivers the
	// event — otherwise a transient DB blip would permanently lose a paid upgrade or
	// a downgrade. Genuinely-ignored event types fall through to 200.
	switch event.Event {
	case "subscription.activated":
		sp, err := parseSubscriptionPayload(event.Payload)
		if err != nil {
			h.log.Error("billing: parse subscription payload", zap.Error(err))
			response.BadRequest(c, "malformed subscription payload")
			return
		}
		if err := h.svc.ActivatePro(ctx, sp.Subscription.Entity.Notes.UserID, sp.Subscription.Entity.ID); err != nil {
			h.log.Error("billing: activate pro", zap.String("subscription_id", sp.Subscription.Entity.ID), zap.Error(err))
			response.InternalError(c)
			return
		}

	case "subscription.charged":
		sp, err := parseSubscriptionPayload(event.Payload)
		if err != nil {
			h.log.Error("billing: parse subscription payload", zap.Error(err))
			response.BadRequest(c, "malformed subscription payload")
			return
		}
		if err := h.svc.RecordCharge(ctx, sp.Subscription.Entity.ID); err != nil {
			h.log.Error("billing: record charge", zap.String("subscription_id", sp.Subscription.Entity.ID), zap.Error(err))
			response.InternalError(c)
			return
		}

	case "subscription.cancelled", "subscription.halted", "subscription.completed":
		sp, err := parseSubscriptionPayload(event.Payload)
		if err != nil {
			h.log.Error("billing: parse subscription payload", zap.Error(err))
			response.BadRequest(c, "malformed subscription payload")
			return
		}
		if err := h.svc.DeactivatePro(ctx, sp.Subscription.Entity.ID); err != nil {
			h.log.Error("billing: deactivate pro", zap.String("subscription_id", sp.Subscription.Entity.ID), zap.Error(err))
			response.InternalError(c)
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// parseSubscriptionPayload unmarshals the nested subscription entity shared by all
// Razorpay subscription webhook events.
func parseSubscriptionPayload(raw json.RawMessage) (*subscriptionPayload, error) {
	var sp subscriptionPayload
	if err := json.Unmarshal(raw, &sp); err != nil {
		return nil, err
	}
	return &sp, nil
}
