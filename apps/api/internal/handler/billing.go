package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	billingsvc "github.com/oyetanishq/yappr/apps/api/internal/service/billing"
	"github.com/oyetanishq/yappr/apps/shared/config"
	"github.com/oyetanishq/yappr/apps/shared/model"
	"github.com/oyetanishq/yappr/apps/shared/pkg/response"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type billingHandler struct {
	svc *billingsvc.Service
	log *zap.Logger
}

func newBillingHandler(client *mongo.Client, log *zap.Logger, cfg *config.Config) *billingHandler {
	svc := billingsvc.New(client, cfg, log)
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
		if err.Error() == fmt.Sprintf("billing: user %s is already on Pro", user.ID) {
			c.JSON(http.StatusConflict, gin.H{"error": "already subscribed to Pro"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active Pro subscription"})
		return
	}

	if err := h.svc.CancelSubscription(ctx, user); err != nil {
		h.log.Error("billing: cancel", zap.String("user_id", user.ID), zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, gin.H{"message": "subscription will be cancelled at end of billing cycle"})
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
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}

	// ── HMAC verification ─────────────────────────────────────────────────────
	sig := c.GetHeader("X-Razorpay-Signature")
	if !h.svc.VerifyWebhookSignature(body, sig) {
		h.log.Warn("billing: webhook signature mismatch")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	var event webhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "malformed payload"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	h.log.Info("billing: webhook received", zap.String("event", event.Event))

	switch event.Event {
	case "subscription.activated", "subscription.charged":
		var sp subscriptionPayload
		if err := json.Unmarshal(event.Payload, &sp); err != nil {
			h.log.Error("billing: parse subscription payload", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "malformed subscription payload"})
			return
		}
		subID := sp.Subscription.Entity.ID
		userID := sp.Subscription.Entity.Notes.UserID

		var activateErr error
		if userID != "" {
			activateErr = h.svc.ActivatePro(ctx, userID, subID)
		} else {
			// Fallback: look up by subscription ID.
			activateErr = h.svc.ActivateProBySubscriptionID(ctx, subID)
		}
		if activateErr != nil {
			h.log.Error("billing: activate pro", zap.String("subscription_id", subID), zap.Error(activateErr))
			// Return 200 so Razorpay doesn't retry — the webhook already fired.
		}

	case "subscription.cancelled", "subscription.halted", "subscription.completed":
		var sp subscriptionPayload
		if err := json.Unmarshal(event.Payload, &sp); err != nil {
			h.log.Error("billing: parse subscription payload", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "malformed subscription payload"})
			return
		}
		subID := sp.Subscription.Entity.ID
		if err := h.svc.DeactivatePro(ctx, subID); err != nil {
			h.log.Error("billing: deactivate pro", zap.String("subscription_id", subID), zap.Error(err))
		}
	}

	// Always return 200 so Razorpay doesn't retry.
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
