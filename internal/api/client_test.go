package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/leffen/devlog-cli/internal/config"
	"github.com/leffen/devlog-cli/internal/git"
)

func TestNewClient(t *testing.T) {
	t.Run("with valid config", func(t *testing.T) {
		cfg := &config.Config{
			APIKey: "test-api-key",
			Server: "https://test.server.com",
		}

		client, err := NewClient(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if client.apiKey != "test-api-key" {
			t.Errorf("expected api key 'test-api-key', got %s", client.apiKey)
		}
		if client.baseURL != "https://test.server.com" {
			t.Errorf("expected base URL 'https://test.server.com', got %s", client.baseURL)
		}
	})

	t.Run("without API key", func(t *testing.T) {
		cfg := &config.Config{
			Server: "https://test.server.com",
		}

		_, err := NewClient(cfg)
		if err == nil {
			t.Error("expected error for missing API key, got nil")
		}
	})
}

func TestCreateEntry(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/api/cli/entries" {
				t.Errorf("expected /api/cli/entries, got %s", r.URL.Path)
			}
			if r.Header.Get("X-API-Key") != "test-key" {
				t.Errorf("expected X-API-Key header 'test-key', got %s", r.Header.Get("X-API-Key"))
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
			}
			if r.Header.Get("User-Agent") != "DevLog-CLI/1.0" {
				t.Errorf("expected User-Agent DevLog-CLI/1.0, got %s", r.Header.Get("User-Agent"))
			}

			// Decode request body
			var req CreateEntryRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("failed to decode request: %v", err)
			}

			if req.Title != "Test Title" {
				t.Errorf("expected title 'Test Title', got %s", req.Title)
			}

			// Send response
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(CreateEntryResponse{
				ID:        "entry-123",
				Title:     req.Title,
				Content:   req.Content,
				Context:   req.Context,
				CreatedAt: "2024-01-15T10:00:00Z",
				Message:   "Entry created successfully",
			})
		}))
		defer server.Close()

		cfg := &config.Config{
			APIKey: "test-key",
			Server: server.URL,
		}

		client, _ := NewClient(cfg)

		resp, err := client.CreateEntry(&CreateEntryRequest{
			Title:   "Test Title",
			Content: "Test content",
			Context: "job",
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if resp.ID != "entry-123" {
			t.Errorf("expected ID 'entry-123', got %s", resp.ID)
		}
		if resp.Title != "Test Title" {
			t.Errorf("expected title 'Test Title', got %s", resp.Title)
		}
	})

	t.Run("API error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: "Invalid request: title is required",
			})
		}))
		defer server.Close()

		cfg := &config.Config{
			APIKey: "test-key",
			Server: server.URL,
		}

		client, _ := NewClient(cfg)

		_, err := client.CreateEntry(&CreateEntryRequest{
			Content: "Test content without title",
		})

		if err == nil {
			t.Error("expected error for bad request, got nil")
		}
		if err.Error() != "API error: Invalid request: title is required" {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestListEntries(t *testing.T) {
	t.Run("successful list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}

			// Check query params
			query := r.URL.Query()
			if query.Get("limit") != "10" {
				t.Errorf("expected limit=10, got %s", query.Get("limit"))
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(ListEntriesResponse{
				Entries: []Entry{
					{
						ID:        "1",
						Title:     "First Entry",
						Content:   "Content 1",
						Context:   "job",
						CreatedAt: "2024-01-15T10:00:00Z",
					},
					{
						ID:        "2",
						Title:     "Second Entry",
						Content:   "Content 2",
						Context:   "project",
						CreatedAt: "2024-01-14T10:00:00Z",
					},
				},
			})
		}))
		defer server.Close()

		cfg := &config.Config{
			APIKey: "test-key",
			Server: server.URL,
		}

		client, _ := NewClient(cfg)

		entries, err := client.ListEntries(10, "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(entries) != 2 {
			t.Errorf("expected 2 entries, got %d", len(entries))
		}
		if entries[0].Title != "First Entry" {
			t.Errorf("expected first entry title 'First Entry', got %s", entries[0].Title)
		}
	})

	t.Run("with filters", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query()
			if query.Get("since") != "2024-01-01" {
				t.Errorf("expected since=2024-01-01, got %s", query.Get("since"))
			}
			if query.Get("project") != "job" {
				t.Errorf("expected project=job, got %s", query.Get("project"))
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(ListEntriesResponse{Entries: []Entry{}})
		}))
		defer server.Close()

		cfg := &config.Config{
			APIKey: "test-key",
			Server: server.URL,
		}

		client, _ := NewClient(cfg)
		client.ListEntries(10, "2024-01-01", "job")
	})
}

func TestBuildGitInfo(t *testing.T) {
	t.Run("with valid context", func(t *testing.T) {
		ctx := &git.Context{
			Remote:  "git@github.com:user/repo.git",
			Project: "repo",
			Branch:  "main",
			Commits: []git.Commit{
				{Hash: "abc123", Message: "test commit"},
			},
		}

		info := BuildGitInfo(ctx)

		if info == nil {
			t.Fatal("expected non-nil GitInfo")
		}
		if info.Remote != ctx.Remote {
			t.Errorf("expected remote %s, got %s", ctx.Remote, info.Remote)
		}
		if info.Project != ctx.Project {
			t.Errorf("expected project %s, got %s", ctx.Project, info.Project)
		}
		if info.Branch != ctx.Branch {
			t.Errorf("expected branch %s, got %s", ctx.Branch, info.Branch)
		}
		if len(info.Commits) != 1 {
			t.Errorf("expected 1 commit, got %d", len(info.Commits))
		}
	})

	t.Run("with nil context", func(t *testing.T) {
		info := BuildGitInfo(nil)
		if info != nil {
			t.Error("expected nil for nil context")
		}
	})
}

func TestEntryStruct(t *testing.T) {
	entry := Entry{
		ID:         "test-id",
		Title:      "Test Entry",
		Content:    "Test content",
		Context:    "job",
		Mood:       "happy",
		Tags:       []string{"go", "testing"},
		Visibility: "private",
		CreatedAt:  "2024-01-15T10:00:00Z",
	}

	if entry.ID != "test-id" {
		t.Error("entry ID mismatch")
	}
	if entry.Title != "Test Entry" {
		t.Error("entry Title mismatch")
	}
	if len(entry.Tags) != 2 {
		t.Error("entry Tags length mismatch")
	}
}
