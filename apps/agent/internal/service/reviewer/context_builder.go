package reviewer

import (
	"fmt"
	"strings"
)

// ── Token budget constants ─────────────────────────────────────────────────────
const (
	maxChangedFilesTokens = 15_000
	approxCharsPerToken   = 4
)

// ReviewContext is the fully assembled context handed to Reviewer.
type ReviewContext struct {
	PRMeta            string
	ChangedFilesBlock string
	RepoSummary       string
	FileCount         int
	TotalAdditions    int
	TotalDeletions    int
}

type ContextBuilder struct{}

func NewContextBuilder() *ContextBuilder {
	return &ContextBuilder{}
}

// Build assembles the context.
func (b *ContextBuilder) Build(prCtx *PRContext) *ReviewContext {
	rc := &ReviewContext{}

	rc.FileCount = len(prCtx.Files)
	rc.TotalAdditions = prCtx.TotalAdditions
	rc.TotalDeletions = prCtx.TotalDeletions

	// ── PR Metadata ──────────────────────────────────────────────────────
	rc.PRMeta = b.buildPRMeta(prCtx)

	// ── Changed Files Block ──────────────────────────────────────────────
	rc.ChangedFilesBlock = b.buildChangedFilesBlock(prCtx, maxChangedFilesTokens*approxCharsPerToken)

	// ── Repo Summary ─────────────────────────────────────────────────────
	rc.RepoSummary = b.buildRepoSummary(prCtx)

	return rc
}

// ── Section builders ──────────────────────────────────────────────────────────

func (b *ContextBuilder) buildPRMeta(prCtx *PRContext) string {
	meta := prCtx.Meta
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Pull Request: %s\n", meta.Title))
	sb.WriteString(fmt.Sprintf("- **Author**: @%s\n", meta.User.Login))
	sb.WriteString(fmt.Sprintf("- **Branch**: `%s` → `%s`\n", meta.Head.Ref, meta.Base.Ref))
	sb.WriteString(fmt.Sprintf("- **Commits**: %s...%s\n", prCtx.BaseSHA[:8], prCtx.HeadSHA[:8]))
	sb.WriteString(fmt.Sprintf("- **Changes**: +%d -%d across %d files\n", prCtx.TotalAdditions, prCtx.TotalDeletions, len(prCtx.Files)))

	if meta.Body != "" {
		body := meta.Body
		if len(body) > 500 {
			body = body[:500] + "...[truncated]"
		}
		sb.WriteString("\n### PR Description:\n")
		sb.WriteString(body)
		sb.WriteString("\n")
	}
	return sb.String()
}

func (b *ContextBuilder) buildChangedFilesBlock(prCtx *PRContext, maxChars int) string {
	var sb strings.Builder
	remaining := maxChars

	for _, cf := range prCtx.Files {
		if remaining <= 0 {
			sb.WriteString("\n> ⚠️ Additional files truncated due to token budget.\n")
			break
		}

		block := b.formatChangedFile(cf)
		if len(block) > remaining {
			// Truncate this file's block
			block = block[:remaining] + "\n... [file truncated due to token budget]\n"
		}
		remaining -= len(block)
		sb.WriteString(block)
	}

	return sb.String()
}

func (b *ContextBuilder) formatChangedFile(cf ChangedFile) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("\n=== FILE: %s (%s) ===\n", cf.Filename, strings.ToUpper(cf.Status)))
	sb.WriteString(fmt.Sprintf("Language: %s | +%d -%d lines\n\n", cf.Language, cf.Additions, cf.Deletions))

	// Diff section
	if cf.Patch != "" {
		sb.WriteString("--- DIFF ---\n")
		sb.WriteString("```diff\n")
		sb.WriteString(cf.Patch)
		sb.WriteString("\n```\n\n")
	}

	return sb.String()
}

func (b *ContextBuilder) buildRepoSummary(prCtx *PRContext) string {
	// Detect primary language from changed files
	langCount := make(map[string]int)
	for _, cf := range prCtx.Files {
		langCount[cf.Language]++
	}
	primaryLang := "unknown"
	maxCount := 0
	for lang, count := range langCount {
		if count > maxCount && lang != "unknown" {
			maxCount = count
			primaryLang = lang
		}
	}

	// Extract repo owner/name
	parts := strings.SplitN(prCtx.Repo, "/", 2)
	repoName := prCtx.Repo
	if len(parts) == 2 {
		repoName = parts[1]
	}

	return fmt.Sprintf("## Repository Summary\n- **Repo**: %s\n- **Primary Language**: %s\n", repoName, primaryLang)
}
