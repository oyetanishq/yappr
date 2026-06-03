package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	sharedgithub "github.com/oyetanishq/yappr/apps/shared/github"
	"go.uber.org/zap"
)

// WebhookService verifies GitHub webhook signatures and dispatches events.
type WebhookService struct {
	secret   string
	ghClient *sharedgithub.Client
	log      *zap.Logger
}

// NewWebhookService creates a WebhookService.
func NewWebhookService(secret string, ghClient *sharedgithub.Client, log *zap.Logger) *WebhookService {
	return &WebhookService{
		secret:   secret,
		ghClient: ghClient,
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
// Extend with more fields as needed.
type pullRequestEvent struct {
	Action      string `json:"action"`
	Number      int    `json:"number"`
	PullRequest struct {
		Title   string `json:"title"`
		State   string `json:"state"`
		HTMLURL string `json:"html_url"`
		User    struct {
			Login string `json:"login"`
		} `json:"user"`
		Head struct {
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
	} `json:"pull_request"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Installation struct {
		ID int64 `json:"id"`
	} `json:"installation"`
}

// installationEvent is a minimal representation of the GitHub installation event.
type installationEvent struct {
	Action       string `json:"action"`
	Installation struct {
		ID      int64 `json:"id"`
		Account struct {
			Login string `json:"login"`
		} `json:"account"`
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
		zap.String("repo", ev.Repository.FullName),
		zap.Int("number", ev.Number),
		zap.String("title", ev.PullRequest.Title),
		zap.String("user", ev.PullRequest.User.Login),
		zap.String("head", ev.PullRequest.Head.Ref),
		zap.String("base", ev.PullRequest.Base.Ref),
		zap.String("url", ev.PullRequest.HTMLURL),
		zap.Int64("install_id", ev.Installation.ID),
	)

	if ev.Action == "opened" {
		comment := fmt.Sprintf(
			"👋 Hey @%s! I've received your PR and I'm processing it now. I'll update you shortly.",
			ev.PullRequest.User.Login,
		)
		if _, err := s.ghClient.PostComment(ctx, ev.Repository.FullName, ev.Number, ev.Installation.ID, comment); err != nil {
			s.log.Error("webhook: failed to post processing comment",
				zap.String("repo", ev.Repository.FullName),
				zap.Int("pr", ev.Number),
				zap.Error(err),
			)
			// Non-fatal — don't fail the webhook because of a comment error.
		} else {
			s.log.Info("webhook: posted processing comment",
				zap.String("repo", ev.Repository.FullName),
				zap.Int("pr", ev.Number),
			)
		}
	}

	// TODO: add your real PR business logic here
	// e.g. save to MongoDB, trigger a code-review job, etc.

	return nil
}
