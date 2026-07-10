// Package reviewer implements the AI-powered pull request code review pipeline.
// It orchestrates cloning the repo and diffing the PR, building structured LLM
// prompts, and posting a rich formatted review back to the PR as a GitHub comment.
package reviewer

import (
	"context"
	"fmt"

	"github.com/oyetanishq/yappr/apps/shared/config"
	sharedgithub "github.com/oyetanishq/yappr/apps/shared/github"
	"github.com/oyetanishq/yappr/apps/shared/model"
	"go.uber.org/zap"
)

// ReviewRequest carries all the information needed to start a review job.
// It is created by the webhook handler and passed to Pipeline.Run().
type ReviewRequest struct {
	// GitHub identifiers
	Repo      string // "owner/repo"
	PRNumber  int
	InstallID int64

	// The comment ID of the "processing..." placeholder posted immediately.
	// The pipeline will EDIT this comment with the final review.
	InitCommentID int64

	// PR metadata passed directly from the webhook payload (avoids one API call)
	PRTitle string
	PRBody  string
	HeadSHA string
	BaseSHA string
	Author  string

	// Per-repo configuration fetched at webhook dispatch time.
	IgnoredPaths      []string          // glob patterns of files to skip in review
	Personality       model.Personality // tone the AI reviewer should use
	EnableArchMapping bool              // whether to run pass B (Pro feature)
}

// ReviewResult holds the structured output from all three LLM passes.
type ReviewResult struct {
	Summary     string // Markdown bullet-point PR summary (Pass A)
	FileChanges string // Per-file analysis markdown (Pass A)
	FlowDiagram string // Raw Mermaid diagram string (Pass B)
	BugReport   string // Full markdown bug section (Pass C)
}

// Pipeline orchestrates the full code review workflow.
type Pipeline struct {
	fetcher *GitHubFetcher
	builder *ContextBuilder
	llm     *Reviewer
	poster  *CommentPoster
	log     *zap.Logger
}

// NewPipeline constructs a Pipeline with all its components wired together.
func NewPipeline(ghClient *sharedgithub.Client, cfg *config.Config, log *zap.Logger) *Pipeline {
	return &Pipeline{
		fetcher: NewGitHubFetcher(ghClient),
		builder: NewContextBuilder(),
		llm:     NewReviewer(cfg, log),
		poster:  NewCommentPoster(ghClient),
		log:     log,
	}
}

// Run executes the full review pipeline for a pull request.
// It is safe to call in a goroutine — all errors are logged internally and a
// fallback error comment is posted if the review fails mid-way.
func (p *Pipeline) Run(ctx context.Context, req ReviewRequest) error {
	p.log.Info("reviewer: starting review pipeline",
		zap.String("repo", req.Repo),
		zap.Int("pr", req.PRNumber),
		zap.String("personality", string(req.Personality)),
		zap.Int("ignored_paths", len(req.IgnoredPaths)),
	)

	// ── Step 1: Fetch full PR context from GitHub API ──────────────────────
	prCtx, err := p.fetcher.Fetch(ctx, req, req.IgnoredPaths)
	if err != nil {
		p.log.Error("reviewer: fetch failed", zap.Error(err))
		_ = p.poster.PostError(ctx, req, fmt.Sprintf("❌ Fetch failed: %v", err))
		return fmt.Errorf("reviewer: fetch: %w", err)
	}

	p.log.Info("reviewer: fetched PR context",
		zap.Int("changed_files", len(prCtx.Files)),
	)

	// ── Step 2: Build structured context for LLM ─────────────────────────
	reviewCtx := p.builder.Build(prCtx)

	// ── Step 3: Multi-pass AI review (personality-aware) ──────────────────
	result, err := p.llm.Review(ctx, reviewCtx, req.Personality, req.EnableArchMapping)
	if err != nil {
		p.log.Error("reviewer: AI review failed", zap.Error(err))
		_ = p.poster.PostError(ctx, req, fmt.Sprintf("❌ AI review failed: %v", err))
		return fmt.Errorf("reviewer: ai: %w", err)
	}

	// ── Step 4: Format and post the final comment ─────────────────────────
	if err := p.poster.Post(ctx, req, reviewCtx, result); err != nil {
		p.log.Error("reviewer: post comment failed", zap.Error(err))
		return fmt.Errorf("reviewer: post: %w", err)
	}

	p.log.Info("reviewer: review complete",
		zap.String("repo", req.Repo),
		zap.Int("pr", req.PRNumber),
	)
	return nil
}
