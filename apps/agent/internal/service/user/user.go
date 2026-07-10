package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/oyetanishq/yappr/apps/shared/config"
	"github.com/oyetanishq/yappr/apps/shared/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

// UserService provides access to user records and handles PR count incrementing.
type UserService struct {
	usersCol         *mongo.Collection
	installationsCol *mongo.Collection
	log              *zap.Logger
}

func NewUserService(client *mongo.Client, cfg *config.Config, log *zap.Logger) *UserService {
	db := client.Database(cfg.Mongo.DB)
	return &UserService{
		usersCol:         db.Collection("users"),
		installationsCol: db.Collection("installations"),
		log:              log,
	}
}

// GetUserByInstallationID finds the installation record by GitHub installation ID,
// then fetches the associated user.
func (s *UserService) GetUserByInstallationID(ctx context.Context, installID int64) (*model.User, error) {
	var inst model.Installation
	err := s.installationsCol.FindOne(ctx, bson.D{{Key: "installation_id", Value: installID}}).Decode(&inst)
	if err != nil {
		return nil, fmt.Errorf("user_svc: find installation %d: %w", installID, err)
	}

	var user model.User
	err = s.usersCol.FindOne(ctx, bson.D{{Key: "_id", Value: inst.UserID}}).Decode(&user)
	if err != nil {
		return nil, fmt.Errorf("user_svc: find user %s: %w", inst.UserID, err)
	}

	return &user, nil
}

// CheckAndIncrementPRCount evaluates if the user has hit their PR limit (for free users).
// If they have not, it increments the monthly counter.
// Returns limitReached=true if they are out of PRs.
func (s *UserService) CheckAndIncrementPRCount(ctx context.Context, userID string) (limitReached bool, err error) {
	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Reset the counter if we've rolled into a new calendar month. Idempotent: the
	// filter only matches when the stored reset date predates this month.
	_, _ = s.usersCol.UpdateOne(ctx,
		bson.D{
			{Key: "_id", Value: userID},
			{Key: "pr_count_reset_at", Value: bson.D{{Key: "$lt", Value: startOfMonth}}},
		},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "pr_count_this_month", Value: 0},
			{Key: "pr_count_reset_at", Value: startOfMonth},
		}}},
	)

	// Fetch to determine Pro status — Pro users are unlimited.
	var user model.User
	if err := s.usersCol.FindOne(ctx, bson.D{{Key: "_id", Value: userID}}).Decode(&user); err != nil {
		return false, fmt.Errorf("user_svc: fetch for limit check: %w", err)
	}
	if user.IsPro() {
		// Unlimited — keep the counter moving for analytics but never cap.
		_, _ = s.usersCol.UpdateOne(ctx,
			bson.D{{Key: "_id", Value: userID}},
			bson.D{{Key: "$inc", Value: bson.D{{Key: "pr_count_this_month", Value: 1}}}},
		)
		return false, nil
	}

	// Free tier: atomically increment only while still under the cap. Doing the check
	// and the increment as a single conditional update closes the TOCTOU race where two
	// concurrent PRs both read count-1 and both slip past the limit.
	res := s.usersCol.FindOneAndUpdate(ctx,
		bson.D{
			{Key: "_id", Value: userID},
			{Key: "pr_count_this_month", Value: bson.D{{Key: "$lt", Value: model.FreePRLimit}}},
		},
		bson.D{{Key: "$inc", Value: bson.D{{Key: "pr_count_this_month", Value: 1}}}},
	)
	if err := res.Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// No document matched the "< limit" filter. The user exists (we just fetched
			// them above), so they are at their monthly cap.
			return true, nil
		}
		return false, fmt.Errorf("user_svc: increment pr count: %w", res.Err())
	}

	return false, nil
}
