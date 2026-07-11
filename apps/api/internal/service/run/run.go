// Package run provides read-only access to PR review run records for the API.
// The agent service is the writer; the API only lists/reads runs scoped to a user.
package run

import (
	"context"
	"fmt"

	"github.com/oyetanishq/yappr/apps/shared/config"
	"github.com/oyetanishq/yappr/apps/shared/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// defaultListLimit caps how many runs the list endpoint returns.
const defaultListLimit = 50

// RunService reads PR review runs from MongoDB.
type RunService struct {
	col *mongo.Collection
	log *zap.Logger
}

// NewRunService creates the service. The pr_runs collection and its indexes are
// created by the agent service (the writer); the API only reads.
func NewRunService(client *mongo.Client, cfg *config.Config, log *zap.Logger) *RunService {
	col := client.Database(cfg.Mongo.DB).Collection("pr_runs")
	return &RunService{col: col, log: log}
}

// ListByUser returns a user's runs, newest first. The large markdown content
// fields are projected out to keep the list payload small — the detail endpoint
// (GetByUser) returns them.
func (s *RunService) ListByUser(ctx context.Context, userID string) ([]model.PRRun, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(defaultListLimit).
		SetProjection(bson.D{
			{Key: "summary", Value: 0},
			{Key: "file_changes", Value: 0},
			{Key: "flow_diagram", Value: 0},
			{Key: "bug_report", Value: 0},
		})

	cur, err := s.col.Find(ctx, bson.D{{Key: "user_id", Value: userID}}, opts)
	if err != nil {
		return nil, fmt.Errorf("run: list by user %s: %w", userID, err)
	}
	defer cur.Close(ctx)

	runs := []model.PRRun{}
	if err := cur.All(ctx, &runs); err != nil {
		return nil, fmt.Errorf("run: decode list %s: %w", userID, err)
	}
	return runs, nil
}

// GetByUser returns a single run by ID, scoped to the owner. Returns
// mongo.ErrNoDocuments if the run doesn't exist or belongs to another user.
func (s *RunService) GetByUser(ctx context.Context, userID, id string) (*model.PRRun, error) {
	var run model.PRRun
	err := s.col.FindOne(ctx, bson.D{
		{Key: "_id", Value: id},
		{Key: "user_id", Value: userID},
	}).Decode(&run)
	if err != nil {
		return nil, err
	}
	return &run, nil
}
