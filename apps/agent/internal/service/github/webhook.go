package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/oyetanishq/yappr/apps/agent/internal/service/reviewer"
	sharedgithub "github.com/oyetanishq/yappr/apps/shared/github"
	"go.uber.org/zap"
)

// WebhookService verifies GitHub webhook signatures and dispatches events.
type WebhookService struct {
	secret   string
	ghClient *sharedgithub.Client
	pipeline *reviewer.Pipeline
	log      *zap.Logger
}

// NewWebhookService creates a WebhookService.
func NewWebhookService(secret string, ghClient *sharedgithub.Client, pipeline *reviewer.Pipeline, log *zap.Logger) *WebhookService {
	return &WebhookService{
		secret:   secret,
		ghClient: ghClient,
		pipeline: pipeline,
		log:      log,
	}
}

// VerifySignature validates the X-Hub-Signature-256 header against the payload.
// GitHub signs every webhook body with HMAC-SHA256 using the webhook secret.
func (s *WebhookService) VerifySignature(payload []byte, sigHeader string) error {
	const prefix = "sha256="
	if !strings.HasPrefix(sigHeader, prefix) {
		return fmt.Errorf("webhook: missing sha256= prefix in signature header")
	}

	got, err := hex.DecodeString(strings.TrimPrefix(sigHeader, prefix))
	if err != nil {
		return fmt.Errorf("webhook: invalid hex in signature header: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write(payload)
	expected := mac.Sum(nil)

	if !hmac.Equal(got, expected) {
		return fmt.Errorf("webhook: signature mismatch")
	}
	return nil
}

// Dispatch routes a verified webhook payload to the appropriate handler.
// eventType is the value of the X-GitHub-Event header.
func (s *WebhookService) Dispatch(ctx context.Context, eventType string, payload []byte) error {
	switch eventType {
	case "pull_request":
		return s.handlePullRequest(ctx, payload)
	case "ping":
		s.log.Info("webhook: ping received — GitHub app configured correctly")
		return nil
	default:
		s.log.Debug("webhook: unhandled event type", zap.String("event", eventType))
		return nil
	}
}

// ── event structs ─────────────────────────────────────────────────────────────

// pullRequestEvent is a minimal representation of the GitHub pull_request event.
type pullRequestEvent struct {
	Action      string `json:"action"`
	Number      int    `json:"number"`
	PullRequest struct {
		Title   string `json:"title"`
		Body    string `json:"body"`
		State   string `json:"state"`
		HTMLURL string `json:"html_url"`
		User    struct {
			Login string `json:"login"`
		} `json:"user"`
		Head struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"base"`
	} `json:"pull_request"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Installation struct {
		ID int64 `json:"id"`
	} `json:"installation"`
}

// ── handlers ──────────────────────────────────────────────────────────────────

func (s *WebhookService) handlePullRequest(ctx context.Context, payload []byte) error {
	var ev pullRequestEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		return fmt.Errorf("webhook: parse pull_request event: %w", err)
	}

	s.log.Info("webhook: pull_request event",
		zap.String("action", ev.Action),
		zap.String("account_name/repo", ev.Repository.FullName),
		zap.Int("pr_number", ev.Number),
		zap.String("pr_title", ev.PullRequest.Title),
		zap.String("user", ev.PullRequest.User.Login),
		zap.String("head_ref", ev.PullRequest.Head.Ref),
		zap.String("base_ref", ev.PullRequest.Base.Ref),
		zap.String("html_url", ev.PullRequest.HTMLURL),
		zap.Int64("github_app_installation_id", ev.Installation.ID),
	)

	// Only trigger review when a PR is first opened.
	// Re-review on new commits (synchronize) is a planned v2 feature.
	if ev.Action != "opened" {
		return nil
	}

	// Post an immediate "processing..." placeholder comment so the developer
	// knows the review is underway. We edit this comment with the full review.
	processingMsg := fmt.Sprintf(
		"## 🤖 Yappr AI Code Review\n\n> ⏳ **Processing PR #%d** — Hey @%s! I've received your PR and I'm analyzing it now.\n>\n> This usually takes **30–60 seconds**. I'll update this comment with:\n> - 📋 PR Summary\n> - 📁 File Change Analysis\n> - 🏗 Architecture Diagram\n> - 🐛 Bug & Edge Case Report (with fixes)\n\n_Please wait..._",
		ev.Number,
		ev.PullRequest.User.Login,
	)

	commentID, err := s.ghClient.PostComment(ctx, ev.Repository.FullName, ev.Number, ev.Installation.ID, processingMsg)
	if err != nil {
		s.log.Error("webhook: failed to post processing comment",
			zap.String("repo", ev.Repository.FullName),
			zap.Int("pr", ev.Number),
			zap.Error(err),
		)
		// Non-fatal — proceed with review even if placeholder comment failed.
		commentID = 0
	} else {
		s.log.Info("webhook: posted processing comment",
			zap.String("repo", ev.Repository.FullName),
			zap.Int("pr", ev.Number),
			zap.Int64("comment_id", commentID),
		)
	}

	// Build the review request from the webhook payload data.
	req := reviewer.ReviewRequest{
		Repo:          ev.Repository.FullName,
		PRNumber:      ev.Number,
		InstallID:     ev.Installation.ID,
		InitCommentID: commentID,
		PRTitle:       ev.PullRequest.Title,
		PRBody:        ev.PullRequest.Body,
		HeadSHA:       ev.PullRequest.Head.SHA,
		BaseSHA:       ev.PullRequest.Base.SHA,
		Author:        ev.PullRequest.User.Login,
	}

	// Launch review in a background goroutine. The webhook handler must return
	// a 200 to GitHub quickly (< 10s) to prevent GitHub from retrying.
	// The pipeline runs with its own 5-minute detached context.
	go func() {
		reviewCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if err := s.pipeline.Run(reviewCtx, req); err != nil {
			s.log.Error("webhook: review pipeline failed",
				zap.String("repo", req.Repo),
				zap.Int("pr", req.PRNumber),
				zap.Error(err),
			)
		}
	}()

	return nil
}
