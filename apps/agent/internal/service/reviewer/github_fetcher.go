package reviewer

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	sharedgithub "github.com/oyetanishq/yappr/apps/shared/github"
)

// PRContext holds everything fetched for a single PR review.
type PRContext struct {
	Meta  *sharedgithub.PRMeta
	Files []ChangedFile

	// Aggregated stats
	TotalAdditions int
	TotalDeletions int

	// Repo identifiers (from ReviewRequest)
	Repo    string
	HeadSHA string
	BaseSHA string
}

// ChangedFile is a file changed in the PR. The unified diff lives in the embedded
// PRFile.Patch; Language is detected from the path.
type ChangedFile struct {
	sharedgithub.PRFile
	Language string // e.g. "go", "typescript", "python", "javascript", "rust"
}

// GitHubFetcher assembles a PRContext: PR metadata comes from the GitHub API,
// the diff comes from a shallow git clone of the repo.
type GitHubFetcher struct {
	client *sharedgithub.Client
}

// NewGitHubFetcher creates a GitHubFetcher.
func NewGitHubFetcher(client *sharedgithub.Client) *GitHubFetcher {
	return &GitHubFetcher{client: client}
}

// cloneDepth bounds how far the shallow fetch reaches when resolving a merge-base.
// 50 covers the vast majority of PRs; beyond it we fall back to a direct
// base..head diff (see cloneAndDiff).
const cloneDepth = 50

// gitSem bounds concurrent clones so a burst of PRs can't exhaust disk/CPU.
var gitSem = make(chan struct{}, maxConcurrentClones())

func maxConcurrentClones() int {
	if n := runtime.NumCPU(); n < 4 {
		return n
	}
	return 4
}

// Fetch retrieves PR metadata (GitHub API) and the changed-file diffs (git clone),
// dropping any file whose path matches an ignoredPaths glob.
func (f *GitHubFetcher) Fetch(ctx context.Context, req ReviewRequest, ignoredPaths []string) (*PRContext, error) {
	// PR metadata (title, author, refs) — one cheap API call.
	meta, err := f.client.GetPRMeta(ctx, req.Repo, req.PRNumber, req.InstallID)
	if err != nil {
		return nil, fmt.Errorf("github fetcher: get PR meta: %w", err)
	}

	// Short-lived installation token, used only as the git clone credential.
	token, err := f.client.InstallationToken(ctx, req.InstallID)
	if err != nil {
		return nil, fmt.Errorf("github fetcher: installation token: %w", err)
	}

	diffs, err := f.cloneAndDiff(ctx, req.Repo, token, req.BaseSHA, req.HeadSHA, req.PRNumber)
	if err != nil {
		return nil, fmt.Errorf("github fetcher: clone and diff: %w", err)
	}

	files := make([]ChangedFile, 0, len(diffs))
	totalAdd, totalDel := 0, 0
	for _, d := range diffs {
		if isIgnored(d.Filename, ignoredPaths) {
			continue
		}
		totalAdd += d.Additions
		totalDel += d.Deletions
		files = append(files, ChangedFile{
			PRFile: sharedgithub.PRFile{
				Filename:         d.Filename,
				Status:           d.Status,
				Additions:        d.Additions,
				Deletions:        d.Deletions,
				Changes:          d.Additions + d.Deletions,
				Patch:            d.Patch,
				PreviousFilename: d.OldFilename,
			},
			Language: detectLanguage(d.Filename),
		})
	}

	return &PRContext{
		Meta:           meta,
		Files:          files,
		TotalAdditions: totalAdd,
		TotalDeletions: totalDel,
		Repo:           req.Repo,
		HeadSHA:        req.HeadSHA,
		BaseSHA:        req.BaseSHA,
	}, nil
}

// gitFileDiff is the parsed per-file result of a git diff.
type gitFileDiff struct {
	Filename    string
	OldFilename string // set for renames
	Status      string // GitHub-style: added/modified/removed/renamed
	Additions   int
	Deletions   int
	Binary      bool
	Patch       string
}

// cloneAndDiff shallow-clones the repo into a throwaway temp dir and returns the
// per-file diff between baseSHA and headSHA. The clone URL embeds the installation
// token; it is never logged, and git errors are redacted before being wrapped.
func (f *GitHubFetcher) cloneAndDiff(ctx context.Context, repo, token, baseSHA, headSHA string, prNumber int) ([]gitFileDiff, error) {
	// Bound concurrency; respect cancellation while waiting for a slot.
	select {
	case gitSem <- struct{}{}:
		defer func() { <-gitSem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Per-PR unique temp dir; MkdirTemp's random suffix prevents collisions between
	// concurrent PRs on the same repo (or the same PR fired twice).
	dir, err := os.MkdirTemp("", fmt.Sprintf("yappr-%s-pr%d-*", sanitizeRepo(repo), prNumber))
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	url := fmt.Sprintf("https://x-access-token:%s@github.com/%s.git", token, repo)

	if _, err := runGit(ctx, dir, "init", "-q"); err != nil {
		return nil, err
	}
	if _, err := runGit(ctx, dir, "remote", "add", "origin", url); err != nil {
		return nil, err
	}
	// Fetch both commits explicitly — a shallow single-branch clone would omit the base.
	if _, err := runGit(ctx, dir, "fetch", "--quiet", "--no-tags", "--depth", strconv.Itoa(cloneDepth), "origin", baseSHA, headSHA); err != nil {
		return nil, err
	}

	// Prefer three-dot (merge-base..head), matching GitHub's "Files changed". If the
	// merge-base isn't within the shallow history (long-lived branch / force-push),
	// fall back to a direct base..head so the review degrades instead of failing.
	diffBase := baseSHA
	if mb, err := runGit(ctx, dir, "merge-base", baseSHA, headSHA); err == nil {
		if s := strings.TrimSpace(string(mb)); s != "" {
			diffBase = s
		}
	}

	nameStatus, err := runGit(ctx, dir, "diff", "--name-status", "-M", "-z", diffBase, headSHA)
	if err != nil {
		return nil, err
	}
	numstat, err := runGit(ctx, dir, "diff", "--numstat", "-M", "-z", diffBase, headSHA)
	if err != nil {
		return nil, err
	}

	files := parseNameStatus(nameStatus)
	stats := parseNumstat(numstat)

	for i := range files {
		if st, ok := stats[files[i].Filename]; ok {
			files[i].Additions = st.additions
			files[i].Deletions = st.deletions
			files[i].Binary = st.binary
		}
		// Skip a per-file patch for binaries and pure renames (mirrors GitHub, which
		// omits those patches).
		if files[i].Binary || (files[i].Status == "renamed" && files[i].Additions == 0 && files[i].Deletions == 0) {
			continue
		}
		patch, err := runGit(ctx, dir, "diff", "-M", "--no-color", diffBase, headSHA, "--", files[i].Filename)
		if err != nil {
			return nil, err
		}
		files[i].Patch = string(patch)
	}

	return files, nil
}

// runGit runs a git command in dir under ctx. GIT_TERMINAL_PROMPT=0 stops a bad
// token from blocking on an interactive credential prompt. stderr is redacted so a
// tokenized remote URL never leaks into a log line or the public error comment.
func runGit(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s: %w: %s", args[0], err, redact(strings.TrimSpace(stderr.String())))
	}
	return stdout.Bytes(), nil
}

// ── Parsing helpers (pure — unit-tested) ───────────────────────────────────────

type numStat struct {
	additions int
	deletions int
	binary    bool
}

// parseNameStatus parses `git diff --name-status -M -z` output. Records are
// NUL-separated: `<status>\0<path>` for A/M/D/T and `R<score>\0<old>\0<new>` for
// renames/copies.
func parseNameStatus(out []byte) []gitFileDiff {
	tokens := splitNUL(out)
	var files []gitFileDiff
	for i := 0; i < len(tokens); {
		status := tokens[i]
		i++
		if status == "" {
			continue
		}
		var oldName, newName string
		if c := status[0]; c == 'R' || c == 'C' {
			if i+1 >= len(tokens) {
				break
			}
			oldName, newName = tokens[i], tokens[i+1]
			i += 2
		} else {
			if i >= len(tokens) {
				break
			}
			newName = tokens[i]
			i++
		}
		files = append(files, gitFileDiff{
			Filename:    newName,
			OldFilename: oldName,
			Status:      mapStatus(status),
		})
	}
	return files
}

// parseNumstat parses `git diff --numstat -M -z` output, keyed by new path. Each
// record is `<adds>\t<dels>\t<path>`; binaries show `-`/`-`; renames emit an empty
// path followed by two extra NUL tokens (old, new).
func parseNumstat(out []byte) map[string]numStat {
	tokens := splitNUL(out)
	stats := make(map[string]numStat)
	for i := 0; i < len(tokens); {
		tok := tokens[i]
		i++
		if tok == "" {
			continue
		}
		parts := strings.SplitN(tok, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		var st numStat
		if parts[0] == "-" || parts[1] == "-" {
			st.binary = true
		} else {
			st.additions, _ = strconv.Atoi(parts[0])
			st.deletions, _ = strconv.Atoi(parts[1])
		}
		path := parts[2]
		if path == "" {
			// rename/copy: the next two tokens are the old and new paths
			if i+1 >= len(tokens) {
				break
			}
			path = tokens[i+1] // new path
			i += 2
		}
		stats[path] = st
	}
	return stats
}

// mapStatus converts a git status letter to the GitHub-style status the rest of
// the pipeline expects.
func mapStatus(gitStatus string) string {
	if gitStatus == "" {
		return "modified"
	}
	switch gitStatus[0] {
	case 'A':
		return "added"
	case 'D':
		return "removed"
	case 'R':
		return "renamed"
	default: // M, T, C, …
		return "modified"
	}
}

var tokenURLRe = regexp.MustCompile(`https://[^/@\s]+:[^/@\s]+@`)

// redact removes any `user:token@` credential from a string so tokenized clone
// URLs never surface in logs or error comments.
func redact(s string) string {
	return tokenURLRe.ReplaceAllString(s, "https://***:***@")
}

func splitNUL(out []byte) []string {
	return strings.Split(string(out), "\x00")
}

func sanitizeRepo(repo string) string {
	return strings.ReplaceAll(repo, "/", "-")
}

// ── Path helpers (reused as-is) ────────────────────────────────────────────────

// detectLanguage returns a normalized language name based on file extension.
func detectLanguage(filename string) string {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".go"):
		return "go"
	case strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx"):
		return "typescript"
	case strings.HasSuffix(lower, ".js") || strings.HasSuffix(lower, ".jsx") || strings.HasSuffix(lower, ".mjs"):
		return "javascript"
	case strings.HasSuffix(lower, ".py"):
		return "python"
	case strings.HasSuffix(lower, ".rs"):
		return "rust"
	case strings.HasSuffix(lower, ".java"):
		return "java"
	case strings.HasSuffix(lower, ".rb"):
		return "ruby"
	case strings.HasSuffix(lower, ".cpp") || strings.HasSuffix(lower, ".cc") || strings.HasSuffix(lower, ".cxx"):
		return "cpp"
	case strings.HasSuffix(lower, ".c"):
		return "c"
	case strings.HasSuffix(lower, ".swift"):
		return "swift"
	case strings.HasSuffix(lower, ".kt") || strings.HasSuffix(lower, ".kts"):
		return "kotlin"
	case strings.HasSuffix(lower, ".sh") || strings.HasSuffix(lower, ".bash"):
		return "shell"
	case strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml"):
		return "yaml"
	case strings.HasSuffix(lower, ".json"):
		return "json"
	case strings.HasSuffix(lower, ".md"):
		return "markdown"
	case strings.HasSuffix(lower, ".sql"):
		return "sql"
	case strings.HasSuffix(lower, ".proto"):
		return "protobuf"
	default:
		return "unknown"
	}
}

// isIgnored reports whether filename should be excluded from the review.
// Each pattern in ignoredPaths is tried as:
//  1. A directory prefix (e.g. "dist/" matches "dist/bundle.js")
//  2. A filepath.Match glob against the full path (e.g. "*.pb.go")
//  3. A filepath.Match glob against the basename (e.g. "*.lock")
func isIgnored(filename string, ignoredPaths []string) bool {
	for _, pattern := range ignoredPaths {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		// Directory prefix check (pattern ends with /)
		if strings.HasSuffix(pattern, "/") {
			if strings.HasPrefix(filename, pattern) {
				return true
			}
			continue
		}
		// Glob match — filepath.Match covers *, ?, [ranges]
		if matched, err := filepath.Match(pattern, filename); err == nil && matched {
			return true
		}
		// Also try matching against just the basename for patterns like "*.lock"
		base := filename[strings.LastIndex(filename, "/")+1:]
		if matched, err := filepath.Match(pattern, base); err == nil && matched {
			return true
		}
	}
	return false
}
