package reviewer

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	sharedgithub "github.com/oyetanishq/yappr/apps/shared/github"
)

// PRContext holds everything fetched from GitHub for a single PR review.
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

// ChangedFile enriches a raw PRFile with the full current file content
// and detected language — the key inputs for AST analysis.
type ChangedFile struct {
	sharedgithub.PRFile
	Content  string // full file content at HeadSHA (empty for deleted files)
	Language string // e.g. "go", "typescript", "python", "javascript", "rust"
}

// GitHubFetcher handles all GitHub API calls needed to assemble PRContext.
type GitHubFetcher struct {
	client *sharedgithub.Client
}

// NewGitHubFetcher creates a GitHubFetcher.
func NewGitHubFetcher(client *sharedgithub.Client) *GitHubFetcher {
	return &GitHubFetcher{client: client}
}

// Fetch retrieves the PR diff, file contents, and metadata.
// For large repos it uses smart sampling: only fetches content for changed files
// plus any nearby files in the same package (for Go blast-radius analysis).
// Files whose paths match any of the ignoredPaths globs are excluded from the review.
func (f *GitHubFetcher) Fetch(ctx context.Context, req ReviewRequest, ignoredPaths []string) (*PRContext, error) {
	// ── 1. Get PR metadata (title, additions, deletions, author) ──────────
	meta, err := f.client.GetPRMeta(ctx, req.Repo, req.PRNumber, req.InstallID)
	if err != nil {
		return nil, fmt.Errorf("github fetcher: get PR meta: %w", err)
	}

	// ── 2. Get list of changed files with their diffs ─────────────────────
	rawFiles, err := f.client.GetPRFiles(ctx, req.Repo, req.PRNumber, req.InstallID)
	if err != nil {
		return nil, fmt.Errorf("github fetcher: get PR files: %w", err)
	}

	// ── 3. Fetch full content for each changed file ───────────────────────
	changedFiles := make([]ChangedFile, 0, len(rawFiles))
	totalAdd, totalDel := 0, 0
	for _, rf := range rawFiles {
		// Skip files that match any configured ignore glob.
		if isIgnored(rf.Filename, ignoredPaths) {
			continue
		}

		totalAdd += rf.Additions
		totalDel += rf.Deletions

		cf := ChangedFile{
			PRFile:   rf,
			Language: detectLanguage(rf.Filename),
		}

		// Fetch content at HeadSHA (skip for deleted files)
		if rf.Status != "removed" {
			content, err := f.client.GetFileContent(ctx, req.Repo, rf.Filename, req.HeadSHA, req.InstallID)
			if err != nil {
				// Non-fatal: proceed with just the diff
				cf.Content = ""
			} else {
				cf.Content = content
			}
		}
		changedFiles = append(changedFiles, cf)
	}

	return &PRContext{
		Meta:           meta,
		Files:          changedFiles,
		TotalAdditions: totalAdd,
		TotalDeletions: totalDel,
		Repo:           req.Repo,
		HeadSHA:        req.HeadSHA,
		BaseSHA:        req.BaseSHA,
	}, nil
}


// FetchBlastRadiusFiles fetches content for files that are NOT in the changed list
// but are candidates for blast-radius analysis (they might call changed symbols).
// Uses smart sampling: only fetches files that match relevant package/import patterns.
func (f *GitHubFetcher) FetchBlastRadiusFiles(
	ctx context.Context,
	req ReviewRequest,
	changedPackages []string,
	changedFilePaths []string,
) ([]ChangedFile, error) {
	// Get the full repo file tree (single API call, returns only paths/SHAs)
	tree, err := f.client.GetRepoTree(ctx, req.Repo, req.HeadSHA, req.InstallID)
	if err != nil {
		return nil, fmt.Errorf("github fetcher: get repo tree: %w", err)
	}

	// Index changed paths for fast exclusion
	changedSet := make(map[string]bool, len(changedFilePaths))
	for _, p := range changedFilePaths {
		changedSet[p] = true
	}

	// Build candidate list: source files not in the changed set
	var candidates []string
	for _, entry := range tree {
		if changedSet[entry.Path] {
			continue
		}
		lang := detectLanguage(entry.Path)
		if lang == "unknown" || lang == "" {
			continue // skip non-source files (images, configs, etc.)
		}
		// For Go: only fetch .go files (not _test.go — those are handled separately)
		candidates = append(candidates, entry.Path)
	}

	// Cap at 200 files to stay well within GitHub's 5000 req/hr rate limit
	// Priority: files in the same directory as changed files come first
	changedDirs := extractDirs(changedFilePaths)
	candidates = prioritizeByDir(candidates, changedDirs)
	if len(candidates) > 200 {
		candidates = candidates[:200]
	}

	// Fetch content for each candidate
	result := make([]ChangedFile, 0, len(candidates))
	for _, path := range candidates {
		content, err := f.client.GetFileContent(ctx, req.Repo, path, req.HeadSHA, req.InstallID)
		if err != nil || content == "" {
			continue
		}

		// Quick pre-filter: check if this file mentions any of the changed packages
		if !mentionsAnyPackage(content, changedPackages) {
			continue
		}

		result = append(result, ChangedFile{
			PRFile: sharedgithub.PRFile{
				Filename: path,
				Status:   "unchanged", // marker for blast-radius files
			},
			Content:  content,
			Language: detectLanguage(path),
		})
	}
	return result, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

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

// extractDirs returns the unique directories of a list of file paths.
func extractDirs(paths []string) []string {
	seen := make(map[string]bool)
	var dirs []string
	for _, p := range paths {
		idx := strings.LastIndex(p, "/")
		if idx < 0 {
			continue // root-level file
		}
		dir := p[:idx]
		if !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

// prioritizeByDir reorders candidates so files in changedDirs come first.
func prioritizeByDir(candidates []string, changedDirs []string) []string {
	dirSet := make(map[string]bool, len(changedDirs))
	for _, d := range changedDirs {
		dirSet[d] = true
	}

	var hi, lo []string
	for _, c := range candidates {
		idx := strings.LastIndex(c, "/")
		dir := ""
		if idx >= 0 {
			dir = c[:idx]
		}
		if dirSet[dir] {
			hi = append(hi, c)
		} else {
			lo = append(lo, c)
		}
	}
	return append(hi, lo...)
}

// mentionsAnyPackage checks whether a file content string references any of the
// given package names. This is a fast string-scan pre-filter before full AST parse.
func mentionsAnyPackage(content string, packages []string) bool {
	if len(packages) == 0 {
		return true // no filter — include everything
	}
	for _, pkg := range packages {
		if strings.Contains(content, pkg) {
			return true
		}
	}
	return false
}

// isIgnored reports whether filename should be excluded from the review.
// Each pattern in ignoredPaths is tried as:
//  1. An exact filepath.Match glob (e.g. "**/*.lock", "*.pb.go")
//  2. A directory prefix (e.g. "dist/" matches "dist/bundle.js")
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

