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
		httpClient: &http.Client{Timeout: 15 * time.Second},
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
		return "", fmt.Errorf("github client: installation token: unexpected status %d", resp.StatusCode)
	}

	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("github client: decode token response: %w", err)
	}
	return body.Token, nil
}

// PostComment posts a comment on a pull request (or any issue — GitHub uses the
// same Issues Comments API for both).
//
//   - ownerRepo: "owner/repo" e.g. "oyetanishq/yappr"
//   - number: PR / issue number
//   - installationID: GitHub App installation ID from the webhook payload
//   - body: comment text (supports GitHub Flavored Markdown)
func (c *Client) PostComment(ctx context.Context, ownerRepo string, number int, installationID int64, body string) error {
	token, err := c.InstallationToken(ctx, installationID)
	if err != nil {
		return err
	}

	type commentRequest struct {
		Body string `json:"body"`
	}
	payload, err := json.Marshal(commentRequest{Body: body})
	if err != nil {
		return fmt.Errorf("github client: marshal comment: %w", err)
	}

	url := fmt.Sprintf("%s/repos/%s/issues/%d/comments", githubAPIBase, ownerRepo, number)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("github client: build comment request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("github client: post comment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("github client: post comment: unexpected status %d", resp.StatusCode)
	}
	return nil
}
