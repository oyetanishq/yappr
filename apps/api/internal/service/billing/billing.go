// Package billing wraps the Razorpay SDK and handles subscription lifecycle
// operations in MongoDB. It is the single source of truth for all plan changes.
package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/oyetanishq/yappr/apps/shared/config"
	"github.com/oyetanishq/yappr/apps/shared/model"
	razorpay "github.com/razorpay/razorpay-go"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// ErrAlreadyPro is returned when a Pro user attempts to subscribe again.
var ErrAlreadyPro = fmt.Errorf("billing: user is already on Pro")

// ErrNotCancelling is returned when a user tries to resume a subscription that is
// not scheduled for cancellation (nothing to undo).
var ErrNotCancelling = fmt.Errorf("billing: no scheduled cancellation to resume")

// Service handles Razorpay subscription management and billing state persistence.
type Service struct {
	rz          *razorpay.Client
	col         *mongo.Collection // users collection
	sessionsCol *mongo.Collection // sessions collection
	rdb         *redis.Client
	cfg         *config.Config
	log         *zap.Logger
}

// New creates a billing Service. It wires up the Razorpay client and users collection.
func New(rdb *redis.Client, mongoClient *mongo.Client, cfg *config.Config, log *zap.Logger) *Service {
	rz := razorpay.NewClient(cfg.Razorpay.KeyID, cfg.Razorpay.KeySecret)
	col := mongoClient.Database(cfg.Mongo.DB).Collection("users")
	sessionsCol := mongoClient.Database(cfg.Mongo.DB).Collection("sessions")
	return &Service{rz: rz, col: col, sessionsCol: sessionsCol, rdb: rdb, cfg: cfg, log: log}
}

// SubscriptionResult is returned by CreateSubscription to the handler.
type SubscriptionResult struct {
	SubscriptionID string `json:"subscription_id"`
	ShortURL       string `json:"short_url"` // Razorpay hosted checkout URL
}

// CreateSubscription creates a new Razorpay subscription for the given user.
// The subscription uses the plan configured in RAZORPAY_PLAN_ID (monthly, INR).
// We return the hosted short_url so the frontend can redirect the user to pay.
func (s *Service) CreateSubscription(ctx context.Context, user *model.User) (*SubscriptionResult, error) {
	if user.IsPro() {
		return nil, ErrAlreadyPro
	}

	// Deduplicate: if the user already has a subscription on file, reuse it instead of
	// creating a second one. A double-click or retry must not spawn two Razorpay
	// subscriptions that each bill monthly. We only create a fresh subscription when
	// there is no live one to reuse.
	if user.RazorpaySubscriptionID != "" {
		existing, err := s.rz.Subscription.Fetch(user.RazorpaySubscriptionID, nil, nil)
		if err != nil {
			s.log.Warn("billing: fetch existing subscription failed; creating a new one",
				zap.String("subscription_id", user.RazorpaySubscriptionID), zap.Error(err))
		} else {
			status, _ := existing["status"].(string)
			switch status {
			case "created", "authenticated", "pending":
				// Checkout is still pending — hand back the same hosted URL.
				shortURL, _ := existing["short_url"].(string)
				return &SubscriptionResult{SubscriptionID: user.RazorpaySubscriptionID, ShortURL: shortURL}, nil
			case "active":
				// Already paid; the activation webhook just hasn't been reflected yet.
				return nil, ErrAlreadyPro
			}
			// halted / cancelled / completed / expired → fall through and create a new one.
		}
	}

	data := map[string]interface{}{
		"plan_id":         s.cfg.Razorpay.PlanID,
		"total_count":     120, // 120 months = 10 years — effectively perpetual monthly
		"quantity":        1,
		"customer_notify": 1,
		"notes": map[string]interface{}{
			"user_id":    user.ID,
			"user_login": user.Login,
		},
	}

	resp, err := s.rz.Subscription.Create(data, nil)
	if err != nil {
		return nil, fmt.Errorf("billing: create subscription: %w", err)
	}

	subID, _ := resp["id"].(string)
	shortURL, _ := resp["short_url"].(string)

	if subID == "" {
		return nil, fmt.Errorf("billing: razorpay returned empty subscription id")
	}

	// Persist the pending subscription id immediately so a subsequent Subscribe call
	// can detect and reuse it (webhook-independent dedup). Refresh the session cache
	// too — RequireAuth reads the user from Redis, so without this a retry would still
	// see an empty id and create a duplicate. Non-fatal on failure: the user can still
	// complete checkout via the returned URL.
	if err := s.setSubscriptionID(ctx, user.ID, subID); err != nil {
		s.log.Error("billing: persist pending subscription id",
			zap.String("user_id", user.ID), zap.String("subscription_id", subID), zap.Error(err))
	} else if err := s.refreshUserSessions(ctx, user.ID); err != nil {
		s.log.Warn("billing: refresh sessions after persisting subscription id",
			zap.String("user_id", user.ID), zap.Error(err))
	}

	return &SubscriptionResult{SubscriptionID: subID, ShortURL: shortURL}, nil
}

// ReactivateSubscription undoes a scheduled cancellation (cancel_at_period_end)
// while the user is still within their paid period, via Razorpay's resume API.
func (s *Service) ReactivateSubscription(ctx context.Context, user *model.User) error {
	if !user.IsPro() || !user.CancelAtPeriodEnd || user.RazorpaySubscriptionID == "" {
		return ErrNotCancelling
	}

	data := map[string]interface{}{"resume_at": "now"}
	if _, err := s.rz.Subscription.Resume(user.RazorpaySubscriptionID, data, nil); err != nil {
		return fmt.Errorf("billing: resume subscription %s: %w", user.RazorpaySubscriptionID, err)
	}

	res, err := s.col.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: user.ID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "cancel_at_period_end", Value: false},
			{Key: "updated_at", Value: time.Now().UTC()},
		}}},
	)
	if err != nil {
		return fmt.Errorf("billing: clear cancel flag for %s: %w", user.ID, err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("billing: reactivate: no user matched id %q", user.ID)
	}

	if err := s.refreshUserSessions(ctx, user.ID); err != nil {
		s.log.Error("billing: refresh user sessions cache", zap.String("user_id", user.ID), zap.Error(err))
	}

	s.log.Info("billing: subscription reactivated",
		zap.String("user_id", user.ID), zap.String("subscription_id", user.RazorpaySubscriptionID))
	return nil
}

// CancelSubscription cancels the active Razorpay subscription at period end
// (cancel_at_cycle_end = true so the user keeps Pro until the paid period expires).
func (s *Service) CancelSubscription(ctx context.Context, user *model.User) error {
	if user.RazorpaySubscriptionID == "" {
		return fmt.Errorf("billing: no active subscription for user %s", user.ID)
	}

	data := map[string]interface{}{
		// "true" means cancel the payment from next month, but keep this month to pro
		// "false" means cancel immediately, also downgrade to free plan right now
		"cancel_at_cycle_end": true,
	}
	_, err := s.rz.Subscription.Cancel(user.RazorpaySubscriptionID, data, nil)
	if err != nil {
		return fmt.Errorf("billing: cancel subscription %s: %w", user.RazorpaySubscriptionID, err)
	}

	userID := user.ID
	_, err = s.col.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "cancel_at_period_end", Value: true},
			{Key: "updated_at", Value: time.Now().UTC()},
		}}},
	)
	if err != nil {
		return fmt.Errorf("billing: set cancel_at_period_end for user %s: %w", userID, err)
	}

	if err := s.refreshUserSessions(ctx, userID); err != nil {
		s.log.Error("billing: refresh user sessions cache", zap.String("user_id", userID), zap.Error(err))
	}

	s.log.Info("billing: subscription cancellation scheduled",
		zap.String("user_id", user.ID),
		zap.String("subscription_id", user.RazorpaySubscriptionID),
	)
	return nil
}

// ActivatePro upgrades the user to Pro in MongoDB. Called from the webhook handler
// on subscription.activated and subscription.charged events.
//
// expiresAt is set to now + 31 days (slightly beyond a calendar month) as a
// safety buffer in case the next charge fires a day late.
func (s *Service) ActivatePro(ctx context.Context, userID, subscriptionID string) error {
	expiresAt := time.Now().UTC().Add(31 * 24 * time.Hour)
	res, err := s.col.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "plan", Value: model.PlanPro},
			{Key: "razorpay_subscription_id", Value: subscriptionID},
			{Key: "plan_expires_at", Value: expiresAt},
			{Key: "cancel_at_period_end", Value: false},
			{Key: "updated_at", Value: time.Now().UTC()},
		}}},
	)
	if err != nil {
		return fmt.Errorf("billing: activate pro for %s: %w", userID, err)
	}
	if res.MatchedCount == 0 {
		// notes.user_id didn't match any user — surface an error so the webhook
		// returns non-2xx and Razorpay retries rather than silently losing the upgrade.
		return fmt.Errorf("billing: activate pro: no user matched id %q (subscription %s)", userID, subscriptionID)
	}

	if err := s.refreshUserSessions(ctx, userID); err != nil {
		s.log.Error("billing: refresh user sessions cache", zap.String("user_id", userID), zap.Error(err))
	}

	s.log.Info("billing: user upgraded to pro", zap.String("user_id", userID), zap.String("subscription_id", subscriptionID))
	return nil
}

// RecordCharge extends Pro access when a recurring charge succeeds
// (subscription.charged). Unlike ActivatePro it is matched by subscription id — not
// user id — and it deliberately does NOT touch cancel_at_period_end, so a renewal
// never silently un-cancels a user. A charge for a subscription we no longer track
// (e.g. a late event arriving after cancellation cleared the id) is ignored
// idempotently, which also blocks out-of-order re-activation of a cancelled user.
func (s *Service) RecordCharge(ctx context.Context, subscriptionID string) error {
	var user model.User
	if err := s.col.FindOne(ctx, bson.D{{Key: "razorpay_subscription_id", Value: subscriptionID}}).Decode(&user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			s.log.Info("billing: charge for untracked subscription, ignoring",
				zap.String("subscription_id", subscriptionID))
			return nil
		}
		return fmt.Errorf("billing: record charge: find user for subscription %s: %w", subscriptionID, err)
	}

	expiresAt := time.Now().UTC().Add(31 * 24 * time.Hour)
	_, err := s.col.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: user.ID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "plan", Value: model.PlanPro},
			{Key: "plan_expires_at", Value: expiresAt},
			{Key: "updated_at", Value: time.Now().UTC()},
		}}},
	)
	if err != nil {
		return fmt.Errorf("billing: record charge for %s: %w", user.ID, err)
	}

	if err := s.refreshUserSessions(ctx, user.ID); err != nil {
		s.log.Error("billing: refresh user sessions cache", zap.String("user_id", user.ID), zap.Error(err))
	}

	s.log.Info("billing: subscription charge recorded",
		zap.String("user_id", user.ID), zap.String("subscription_id", subscriptionID))
	return nil
}

// DeactivatePro downgrades the user to Free. Called from the webhook on
// subscription.cancelled / halted / completed. It clears the stored subscription id
// so a late subscription.charged for the same (now-terminated) subscription cannot
// re-activate the user. Idempotent: a duplicate terminal event for an
// already-cleared subscription is a no-op.
func (s *Service) DeactivatePro(ctx context.Context, subscriptionID string) error {
	// Find the user by subscription id first, so we can key the update on _id and
	// refresh their session cache afterwards.
	var user model.User
	if err := s.col.FindOne(ctx, bson.D{{Key: "razorpay_subscription_id", Value: subscriptionID}}).Decode(&user); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Already downgraded / unknown subscription — nothing to do.
			s.log.Info("billing: deactivate for untracked subscription, ignoring",
				zap.String("subscription_id", subscriptionID))
			return nil
		}
		return fmt.Errorf("billing: deactivate pro: find user for subscription %s: %w", subscriptionID, err)
	}

	_, err := s.col.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: user.ID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "plan", Value: model.PlanFree},
			{Key: "plan_expires_at", Value: nil},
			{Key: "cancel_at_period_end", Value: false},
			{Key: "razorpay_subscription_id", Value: ""},
			{Key: "updated_at", Value: time.Now().UTC()},
		}}},
	)
	if err != nil {
		return fmt.Errorf("billing: deactivate pro for %s: %w", user.ID, err)
	}

	if err := s.refreshUserSessions(ctx, user.ID); err != nil {
		s.log.Error("billing: refresh user sessions cache", zap.String("user_id", user.ID), zap.Error(err))
	}

	s.log.Info("billing: user downgraded to free",
		zap.String("user_id", user.ID), zap.String("subscription_id", subscriptionID))
	return nil
}

// VerifyWebhookSignature validates the Razorpay webhook HMAC-SHA256 signature.
// Razorpay signs the raw body with the webhook secret using SHA-256.
func (s *Service) VerifyWebhookSignature(payload []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(s.cfg.Razorpay.WebhookSecret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (s *Service) setSubscriptionID(ctx context.Context, userID, subscriptionID string) error {
	_, err := s.col.UpdateOne(ctx,
		bson.D{{Key: "_id", Value: userID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "razorpay_subscription_id", Value: subscriptionID},
			{Key: "updated_at", Value: time.Now().UTC()},
		}}},
	)
	return err
}

func (s *Service) refreshUserSessions(ctx context.Context, userID string) error {
	// 1. Fetch updated user document
	var user model.User
	if err := s.col.FindOne(ctx, bson.D{{Key: "_id", Value: userID}}).Decode(&user); err != nil {
		return fmt.Errorf("find user %s: %w", userID, err)
	}

	userJSON, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("marshal user %s: %w", userID, err)
	}

	// 2. Find all active sessions for this user
	cur, err := s.sessionsCol.Find(ctx, bson.D{{Key: "user_id", Value: userID}})
	if err != nil {
		return fmt.Errorf("find sessions for %s: %w", userID, err)
	}
	defer cur.Close(ctx)

	var sessions []model.Session
	if err := cur.All(ctx, &sessions); err != nil {
		return fmt.Errorf("decode sessions for %s: %w", userID, err)
	}

	// 3. Update Redis cache for each session
	for _, session := range sessions {
		// Redis EX is managed on creation, KeepTTL preserves the original expiration
		if err := s.rdb.Set(ctx, "session:"+session.ID, userJSON, redis.KeepTTL).Err(); err != nil {
			s.log.Warn("billing: failed to update redis session", zap.String("session_id", session.ID), zap.Error(err))
		}
	}

	return nil
}
