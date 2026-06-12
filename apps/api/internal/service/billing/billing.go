// Package billing wraps the Razorpay SDK and handles subscription lifecycle
// operations in MongoDB. It is the single source of truth for all plan changes.
package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
		return nil, fmt.Errorf("billing: user %s is already on Pro", user.ID)
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

	// Persist the subscription ID on the user document immediately so the
	// webhook handler can match it back to a user.
	if err := s.setSubscriptionID(ctx, user.ID, subID); err != nil {
		s.log.Error("billing: failed to persist subscription id", zap.String("user_id", user.ID), zap.Error(err))
		// Non-fatal — webhook will still fire and activate the plan.
	}

	return &SubscriptionResult{SubscriptionID: subID, ShortURL: shortURL}, nil
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
	_, err := s.col.UpdateOne(ctx,
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

	if err := s.refreshUserSessions(ctx, userID); err != nil {
		s.log.Error("billing: refresh user sessions cache", zap.String("user_id", userID), zap.Error(err))
	}

	s.log.Info("billing: user upgraded to pro", zap.String("user_id", userID))
	return nil
}

// DeactivatePro downgrades the user to Free. Called from the webhook on
// subscription.cancelled or subscription.halted events.
func (s *Service) DeactivatePro(ctx context.Context, subscriptionID string) error {
	_, err := s.col.UpdateOne(ctx,
		bson.D{{Key: "razorpay_subscription_id", Value: subscriptionID}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "plan", Value: model.PlanFree},
			{Key: "plan_expires_at", Value: nil},
			{Key: "cancel_at_period_end", Value: false},
			{Key: "updated_at", Value: time.Now().UTC()},
		}}},
	)
	if err != nil {
		return fmt.Errorf("billing: deactivate pro for subscription %s: %w", subscriptionID, err)
	}

	// We need the userID to refresh sessions. Find the user first.
	var user model.User
	if err := s.col.FindOne(ctx, bson.D{{Key: "razorpay_subscription_id", Value: subscriptionID}}).Decode(&user); err == nil {
		if err := s.refreshUserSessions(ctx, user.ID); err != nil {
			s.log.Error("billing: refresh user sessions cache", zap.String("user_id", user.ID), zap.Error(err))
		}
	}

	s.log.Info("billing: user downgraded to free", zap.String("subscription_id", subscriptionID))
	return nil
}

// IncrementPRCount atomically increments a user's monthly PR review counter.
// If the calendar month has rolled over it resets the counter first.
// Returns the updated count and whether the limit was reached (for free users).
func (s *Service) IncrementPRCount(ctx context.Context, userID string) (int, error) {
	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Attempt to reset if the reset date is in a previous month.
	_, _ = s.col.UpdateOne(ctx,
		bson.D{
			{Key: "_id", Value: userID},
			{Key: "pr_count_reset_at", Value: bson.D{{Key: "$lt", Value: startOfMonth}}},
		},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "pr_count_this_month", Value: 0},
			{Key: "pr_count_reset_at", Value: startOfMonth},
		}}},
	)

	// Now increment.
	result := s.col.FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: userID}},
		bson.D{{Key: "$inc", Value: bson.D{{Key: "pr_count_this_month", Value: 1}}}},
		nil,
	)
	if result.Err() != nil {
		return 0, fmt.Errorf("billing: increment pr count for %s: %w", userID, result.Err())
	}

	var updated model.User
	if err := result.Decode(&updated); err != nil {
		return 0, fmt.Errorf("billing: decode user after increment: %w", err)
	}

	newCount := updated.PRCountThisMonth + 1 // FindOneAndUpdate returns BEFORE doc by default
	return newCount, nil
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
