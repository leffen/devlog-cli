package git

import (
	"strings"
	"testing"
)

func TestExtractProjectName(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "SSH format with .git",
			url:      "git@github.com:user/my-repo.git",
			expected: "my-repo",
		},
		{
			name:     "SSH format without .git",
			url:      "git@github.com:user/my-repo",
			expected: "my-repo",
		},
		{
			name:     "HTTPS format with .git",
			url:      "https://github.com/user/my-repo.git",
			expected: "my-repo",
		},
		{
			name:     "HTTPS format without .git",
			url:      "https://github.com/user/my-repo",
			expected: "my-repo",
		},
		{
			name:     "GitLab SSH",
			url:      "git@gitlab.com:group/subgroup/project.git",
			expected: "project",
		},
		{
			name:     "GitLab HTTPS",
			url:      "https://gitlab.com/group/subgroup/project.git",
			expected: "project",
		},
		{
			name:     "Bitbucket SSH",
			url:      "git@bitbucket.org:user/repo.git",
			expected: "repo",
		},
		{
			name:     "Self-hosted GitLab",
			url:      "git@git.company.com:team/awesome-project.git",
			expected: "awesome-project",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractProjectName(tc.url)
			if result != tc.expected {
				t.Errorf("ExtractProjectName(%q) = %q, want %q", tc.url, result, tc.expected)
			}
		})
	}
}

func TestFormatCommitsMarkdown(t *testing.T) {
	commits := []Commit{
		{Hash: "abc1234", Message: "feat: add new feature"},
		{Hash: "def5678", Message: "fix: resolve bug"},
		{Hash: "ghi9012", Message: "docs: update README"},
	}

	t.Run("with hashes", func(t *testing.T) {
		result := FormatCommitsMarkdown(commits, true)

		if !strings.Contains(result, "`abc1234`") {
			t.Error("expected hash abc1234 in output")
		}
		if !strings.Contains(result, "feat: add new feature") {
			t.Error("expected commit message in output")
		}
		if !strings.Contains(result, "- `") {
			t.Error("expected markdown list format with backticks")
		}

		lines := strings.Split(strings.TrimSpace(result), "\n")
		if len(lines) != 3 {
			t.Errorf("expected 3 lines, got %d", len(lines))
		}
	})

	t.Run("without hashes", func(t *testing.T) {
		result := FormatCommitsMarkdown(commits, false)

		if strings.Contains(result, "`abc1234`") {
			t.Error("expected no hash in output")
		}
		if !strings.Contains(result, "- feat: add new feature") {
			t.Error("expected commit message with dash prefix")
		}

		lines := strings.Split(strings.TrimSpace(result), "\n")
		if len(lines) != 3 {
			t.Errorf("expected 3 lines, got %d", len(lines))
		}
	})

	t.Run("empty commits", func(t *testing.T) {
		result := FormatCommitsMarkdown([]Commit{}, true)
		if result != "" {
			t.Errorf("expected empty string for empty commits, got %q", result)
		}
	})
}

func TestIsInGitRepo(t *testing.T) {
	// This test runs in the actual project directory which is a git repo
	if !IsInGitRepo() {
		t.Skip("test must be run from within a git repository")
	}

	// The function should return true since we're in the devlog-cli repo
	result := IsInGitRepo()
	if !result {
		t.Error("expected IsInGitRepo to return true in project directory")
	}
}

func TestGetContext(t *testing.T) {
	if !IsInGitRepo() {
		t.Skip("test must be run from within a git repository")
	}

	ctx, err := GetContext()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have remote and branch at minimum
	if ctx.Remote == "" {
		t.Error("expected non-empty remote URL")
	}
	if ctx.Branch == "" {
		t.Error("expected non-empty branch name")
	}
	if ctx.Project == "" {
		t.Error("expected non-empty project name")
	}

	// Project should be extracted from remote
	if ctx.Project != "devlog-cli" {
		t.Errorf("expected project 'devlog-cli', got %q", ctx.Project)
	}
}

func TestCommitStruct(t *testing.T) {
	commit := Commit{
		Hash:    "abc1234",
		Message: "test commit",
		Author:  "Test User",
		Date:    "2024-01-15",
	}

	if commit.Hash != "abc1234" {
		t.Errorf("expected hash abc1234, got %s", commit.Hash)
	}
	if commit.Message != "test commit" {
		t.Errorf("expected message 'test commit', got %s", commit.Message)
	}
	if commit.Author != "Test User" {
		t.Errorf("expected author 'Test User', got %s", commit.Author)
	}
	if commit.Date != "2024-01-15" {
		t.Errorf("expected date '2024-01-15', got %s", commit.Date)
	}
}

func TestContextStruct(t *testing.T) {
	ctx := Context{
		Remote:  "git@github.com:user/repo.git",
		Project: "repo",
		Branch:  "main",
		Commits: []Commit{
			{Hash: "abc", Message: "test"},
		},
	}

	if ctx.Remote != "git@github.com:user/repo.git" {
		t.Error("context remote mismatch")
	}
	if ctx.Project != "repo" {
		t.Error("context project mismatch")
	}
	if ctx.Branch != "main" {
		t.Error("context branch mismatch")
	}
	if len(ctx.Commits) != 1 {
		t.Error("context commits length mismatch")
	}
}
