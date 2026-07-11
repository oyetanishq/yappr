// Package run persists PR review run records to MongoDB. The agent is the writer:
// it creates a run when a PR is opened and updates it as the review pipeline
// progresses. The API service reads these records to power the dashboard history.
package run

import (
	"context"
	"fmt"
	"time"

	"github.com/oyetanishq/yappr/apps/shared/config"
	"github.com/oyetanishq/yappr/apps/shared/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// RunService writes PR review run records.
type RunService struct {
	col *mongo.Collection
	log *zap.Logger
}

// NewRunService creates the service and ensures the list-query index exists.
func NewRunService(client *mongo.Client, cfg *config.Config, log *zap.Logger) *RunService {
	col := client.Database(cfg.Mongo.DB).Collection("pr_runs")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Index for the dashboard list query: a user's runs, newest first.
	if _, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
		Options: options.Index().SetName("user_id_created_at"),
	}); err != nil {
		log.Warn("mongo pr_runs index: user_id/created_at", zap.Error(err))
	}

	return &RunService{col: col, log: log}
}

// Create inserts a new run record. The caller sets Status (processing or
// limit_reached) and all metadata; ID/CreatedAt/UpdatedAt are filled here.
// Returns the generated run ID.
func (s *RunService) Create(ctx context.Context, run model.PRRun) (string, error) {
	now := time.Now().UTC()
	run.ID = primitive.NewObjectID().Hex()
	run.CreatedAt = now
	run.UpdatedAt = now

	if _, err := s.col.InsertOne(ctx, run); err != nil {
		return "", fmt.Errorf("run: create: %w", err)
	}
	return run.ID, nil
}

// CompletionData carries the review output persisted when a run finishes. It uses
// primitives so this package doesn't import the reviewer package (avoids a cycle).
type CompletionData struct {
	FilesChanged int
	Additions    int
	Deletions    int
	Summary      string
	FileChanges  string
	FlowDiagram  string
	BugReport    string
}

// MarkCompleted transitions a run to "completed" and stores its stats + content.
func (s *RunService) MarkCompleted(ctx context.Context, id string, data CompletionData) error {
	now := time.Now().UTC()
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "status", Value: model.RunStatusCompleted},
		{Key: "files_changed", Value: data.FilesChanged},
		{Key: "additions", Value: data.Additions},
		{Key: "deletions", Value: data.Deletions},
		{Key: "summary", Value: data.Summary},
		{Key: "file_changes", Value: data.FileChanges},
		{Key: "flow_diagram", Value: data.FlowDiagram},
		{Key: "bug_report", Value: data.BugReport},
		{Key: "completed_at", Value: now},
		{Key: "updated_at", Value: now},
	}}}
	if _, err := s.col.UpdateByID(ctx, id, update); err != nil {
		return fmt.Errorf("run: mark completed %s: %w", id, err)
	}
	return nil
}

// MarkFailed transitions a run to "failed" with an error message.
func (s *RunService) MarkFailed(ctx context.Context, id, errMsg string) error {
	now := time.Now().UTC()
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "status", Value: model.RunStatusFailed},
		{Key: "error", Value: errMsg},
		{Key: "completed_at", Value: now},
		{Key: "updated_at", Value: now},
	}}}
	if _, err := s.col.UpdateByID(ctx, id, update); err != nil {
		return fmt.Errorf("run: mark failed %s: %w", id, err)
	}
	return nil
}
