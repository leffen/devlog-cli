package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/leffen/devlog-cli/internal/config"
	"github.com/leffen/devlog-cli/internal/git"
)

// Client is the DevLog API client
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new API client
func NewClient(cfg *config.Config) (*Client, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key not configured. Run 'devlog config set api-key YOUR_KEY'")
	}

	return &Client{
		baseURL: cfg.Server,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Entry represents a DevLog entry
type Entry struct {
	ID         string   `json:"id,omitempty"`
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Context    string   `json:"context,omitempty"` // job, project, fun
	Mood       string   `json:"mood,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Visibility string   `json:"visibility,omitempty"` // private, public
	Source     string   `json:"source,omitempty"`     // cli, claude-code, cursor, etc.
	Repository string   `json:"repository,omitempty"` // repository name
	Git        *GitInfo `json:"git,omitempty"`
	CreatedAt  string   `json:"createdAt,omitempty"`
	UpdatedAt  string   `json:"updatedAt,omitempty"`
}

// GitInfo represents git context in an entry
type GitInfo struct {
	Remote  string       `json:"remote,omitempty"`
	Project string       `json:"project,omitempty"`
	Branch  string       `json:"branch,omitempty"`
	Commits []git.Commit `json:"commits,omitempty"`
}

// CreateEntryRequest is the request body for creating an entry
type CreateEntryRequest struct {
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Context    string   `json:"context,omitempty"` // job, project, fun
	Mood       string   `json:"mood,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	Visibility string   `json:"visibility,omitempty"`
	Source     string   `json:"source,omitempty"`     // cli, claude-code, cursor, etc.
	Repository string   `json:"repository,omitempty"` // repository name
	Git        *GitInfo `json:"git,omitempty"`
}

// CreateEntryResponse is the response from creating an entry
type CreateEntryResponse struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Context    string   `json:"context"`
	Mood       string   `json:"mood"`
	Tags       []string `json:"tags"`
	Visibility string   `json:"visibility"`
	Source     string   `json:"source"`
	Repository string   `json:"repository"`
	CreatedAt  string   `json:"createdAt"`
	Message    string   `json:"message"`
}

// ListEntriesResponse is the response from listing entries
type ListEntriesResponse struct {
	Entries []Entry `json:"entries"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error string `json:"error"`
}

// doRequest performs an HTTP request with API key authentication
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "DevLog-CLI/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// CreateEntry creates a new entry
func (c *Client) CreateEntry(req *CreateEntryRequest) (*CreateEntryResponse, error) {
	resp, err := c.doRequest("POST", "/api/cli/entries", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("API error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result CreateEntryResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ListEntries lists recent entries
func (c *Client) ListEntries(limit int, since, project string) ([]Entry, error) {
	path := fmt.Sprintf("/api/cli/entries?limit=%d", limit)
	if since != "" {
		path += "&since=" + since
	}
	if project != "" {
		path += "&project=" + project
	}

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
			return nil, fmt.Errorf("API error: %s", errResp.Error)
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result ListEntriesResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Entries, nil
}

// BuildGitInfo creates GitInfo from git context
func BuildGitInfo(ctx *git.Context) *GitInfo {
	if ctx == nil {
		return nil
	}

	return &GitInfo{
		Remote:  ctx.Remote,
		Project: ctx.Project,
		Branch:  ctx.Branch,
		Commits: ctx.Commits,
	}
}
