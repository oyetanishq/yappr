// Package github provides a GitHub App client for authenticating as an
// installation and interacting with the GitHub API (posting PR comments, etc.).
// It can be imported by any app in this workspace.
package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const githubAPIBase = "https://api.github.com"

// Client authenticates as a GitHub App and interacts with the GitHub API.
type Client struct {
	appID      string
	privateKey string // base64-encoded PEM private key (set via GITHUB_APP_PRIVATE_KEY env var)
	httpClient *http.Client
}

// NewClient creates a Client using the App ID and base64-encoded PEM private key.
func NewClient(appID, privateKey string) *Client {
	return &Client{
		appID:      appID,
		privateKey: privateKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// GenerateAppJWT creates a short-lived JWT signed with the GitHub App's private key.
// GitHub requires this to exchange for an installation access token.
func (c *Client) GenerateAppJWT() (string, error) {
	decodedPEM, err := base64.StdEncoding.DecodeString(c.privateKey)
	if err != nil {
		return "", fmt.Errorf("github client: decode base64 private key: %w", err)
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(decodedPEM)
	if err != nil {
		return "", fmt.Errorf("github client: parse private key: %w", err)
	}

	now := time.Now()
	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now.Add(-60 * time.Second)), // allow for clock skew
		ExpiresAt: jwt.NewNumericDate(now.Add(9 * time.Minute)),   // max 10 min
		Issuer:    c.appID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("github client: sign JWT: %w", err)
	}
	return signed, nil
}

// InstallationToken exchanges the App JWT for a short-lived installation access token.
func (c *Client) InstallationToken(ctx context.Context, installationID int64) (string, error) {
	appJWT, err := c.GenerateAppJWT()
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/app/installations/%d/access_tokens", githubAPIBase, installationID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", fmt.Errorf("github client: build token request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+appJWT)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("github client: request installation token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("github client: installation token: unexpected status %d: %s", resp.StatusCode, body)
	}

	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("github client: decode token response: %w", err)
	}
	return body.Token, nil
}

// doWithToken is a helper that authenticates and executes an HTTP request using
// an installation access token, returning the raw response body.
func (c *Client) doWithToken(ctx context.Context, installationID int64, method, url string, payload []byte) ([]byte, int, error) {
	token, err := c.InstallationToken(ctx, installationID)
	if err != nil {
		return nil, 0, err
	}

	var bodyReader io.Reader
	if payload != nil {
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("github client: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("github client: execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("github client: read response: %w", err)
	}
	return respBody, resp.StatusCode, nil
}

// ── Comment APIs ──────────────────────────────────────────────────────────────

// PostComment posts a comment on a pull request and returns the created comment's ID.
//
//   - ownerRepo: "owner/repo" e.g. "oyetanishq/yappr"
//   - number: PR / issue number
//   - installationID: GitHub App installation ID from the webhook payload
//   - body: comment text (supports GitHub Flavored Markdown)
func (c *Client) PostComment(ctx context.Context, ownerRepo string, number int, installationID int64, body string) (int64, error) {
	type commentRequest struct {
		Body string `json:"body"`
	}
	payload, err := json.Marshal(commentRequest{Body: body})
	if err != nil {
		return 0, fmt.Errorf("github client: marshal comment: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/issues/%d/comments", githubAPIBase, ownerRepo, number)
	respBody, status, err := c.doWithToken(ctx, installationID, http.MethodPost, url, payload)
	if err != nil {
		return 0, err
	}
	if status != http.StatusCreated {
		return 0, fmt.Errorf("github client: post comment: unexpected status %d: %s", status, respBody)
	}

	var created struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return 0, fmt.Errorf("github client: decode comment response: %w", err)
	}
	return created.ID, nil
}

// UpdateComment edits an existing issue/PR comment body.
func (c *Client) UpdateComment(ctx context.Context, ownerRepo string, commentID int64, installationID int64, body string) error {
	type commentRequest struct {
		Body string `json:"body"`
	}
	payload, err := json.Marshal(commentRequest{Body: body})
	if err != nil {
		return fmt.Errorf("github client: marshal comment update: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/issues/comments/%d", githubAPIBase, ownerRepo, commentID)
	respBody, status, err := c.doWithToken(ctx, installationID, http.MethodPatch, url, payload)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("github client: update comment: unexpected status %d: %s", status, respBody)
	}
	return nil
}

// ── Pull Request APIs ─────────────────────────────────────────────────────────

// PRMeta contains metadata about a pull request.
type PRMeta struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	State     string `json:"state"`
	HTMLURL   string `json:"html_url"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
	Head struct {
		SHA string `json:"sha"`
		Ref string `json:"ref"`
	} `json:"head"`
	Base struct {
		SHA string `json:"sha"`
		Ref string `json:"ref"`
	} `json:"base"`
}

// GetPRMeta fetches full pull request metadata.
func (c *Client) GetPRMeta(ctx context.Context, ownerRepo string, prNumber int, installationID int64) (*PRMeta, error) {
	url := fmt.Sprintf("%s/repos/%s/pulls/%d", githubAPIBase, ownerRepo, prNumber)
	body, status, err := c.doWithToken(ctx, installationID, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("github client: get PR meta: unexpected status %d: %s", status, body)
	}

	var meta PRMeta
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, fmt.Errorf("github client: decode PR meta: %w", err)
	}
	return &meta, nil
}

// PRFile represents a file changed in a pull request.
type PRFile struct {
	Filename    string `json:"filename"`
	Status      string `json:"status"` // added, modified, removed, renamed
	Additions   int    `json:"additions"`
	Deletions   int    `json:"deletions"`
	Changes     int    `json:"changes"`
	Patch       string `json:"patch"` // unified diff hunk
	PreviousFilename string `json:"previous_filename"` // set when renamed
}

// GetPRFiles returns the list of files changed in a pull request with their diffs.
// GitHub paginates at 300 files max; we fetch all pages up to that limit.
func (c *Client) GetPRFiles(ctx context.Context, ownerRepo string, prNumber int, installationID int64) ([]PRFile, error) {
	var allFiles []PRFile
	page := 1
	for {
		url := fmt.Sprintf("%s/repos/%s/pulls/%d/files?per_page=100&page=%d", githubAPIBase, ownerRepo, prNumber, page)
		body, status, err := c.doWithToken(ctx, installationID, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		if status != http.StatusOK {
			return nil, fmt.Errorf("github client: get PR files: unexpected status %d: %s", status, body)
		}

		var files []PRFile
		if err := json.Unmarshal(body, &files); err != nil {
			return nil, fmt.Errorf("github client: decode PR files: %w", err)
		}
		allFiles = append(allFiles, files...)
		if len(files) < 100 {
			break // last page
		}
		page++
	}
	return allFiles, nil
}

// GetFileContent fetches the raw content of a file at a specific ref (branch/SHA).
// Returns an empty string (not an error) if the file does not exist at that ref.
func (c *Client) GetFileContent(ctx context.Context, ownerRepo, path, ref string, installationID int64) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", githubAPIBase, ownerRepo, path, ref)
	body, status, err := c.doWithToken(ctx, installationID, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	if status == http.StatusNotFound {
		return "", nil // file doesn't exist at this ref — not an error
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("github client: get file content: status %d for %s: %s", status, path, body)
	}

	var resp struct {
		Content  string `json:"content"`  // base64 encoded, may contain newlines
		Encoding string `json:"encoding"` // "base64"
		Message  string `json:"message"`  // set if it's a directory or error
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("github client: decode file content: %w", err)
	}
	if resp.Encoding == "" {
		// Likely a directory listing or too-large file response
		return "", fmt.Errorf("github client: file %s: unexpected response (possibly a directory)", path)
	}

	// GitHub returns base64 with newlines embedded — strip them before decoding
	b64clean := ""
	for _, ch := range resp.Content {
		if ch != '\n' && ch != '\r' {
			b64clean += string(ch)
		}
	}

	decoded, err := base64.StdEncoding.DecodeString(b64clean)
	if err != nil {
		return "", fmt.Errorf("github client: decode file base64 for %s: %w", path, err)
	}
	return string(decoded), nil
}

// ── Repository Tree API ───────────────────────────────────────────────────────

// TreeEntry is a single item (file or directory) in a repository tree.
type TreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"` // "blob" (file) or "tree" (dir)
	Size int    `json:"size"`
	SHA  string `json:"sha"`
}

// GetRepoTree returns the full recursive file tree of a repository at a given SHA.
// This is a single API call that returns all file paths without content — very efficient
// for building a list of files to selectively fetch for blast-radius analysis.
func (c *Client) GetRepoTree(ctx context.Context, ownerRepo, treeSHA string, installationID int64) ([]TreeEntry, error) {
	url := fmt.Sprintf("%s/repos/%s/git/trees/%s?recursive=1", githubAPIBase, ownerRepo, treeSHA)
	body, status, err := c.doWithToken(ctx, installationID, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("github client: get repo tree: unexpected status %d: %s", status, body)
	}

	var resp struct {
		Tree     []TreeEntry `json:"tree"`
		Truncated bool       `json:"truncated"` // true if repo has >100k items
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("github client: decode repo tree: %w", err)
	}

	// Filter to only files (blobs), not directories
	files := make([]TreeEntry, 0, len(resp.Tree))
	for _, entry := range resp.Tree {
		if entry.Type == "blob" {
			files = append(files, entry)
		}
	}
	return files, nil
}
