package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// Context represents git context for the current directory
type Context struct {
	Remote   string   `json:"remote,omitempty"`
	Project  string   `json:"project,omitempty"`
	Branch   string   `json:"branch,omitempty"`
	Commits  []Commit `json:"commits,omitempty"`
}

// Commit represents a git commit
type Commit struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
	Author  string `json:"author,omitempty"`
	Date    string `json:"date,omitempty"`
}

// IsInGitRepo checks if the current directory is in a git repository
func IsInGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}

// GetRemoteURL returns the remote URL for origin
func GetRemoteURL() (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get remote URL: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		// Might be in detached HEAD state
		cmd = exec.Command("git", "rev-parse", "--short", "HEAD")
		out, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to get HEAD: %w", err)
		}
		return "HEAD:" + strings.TrimSpace(string(out)), nil
	}
	return branch, nil
}

// ExtractProjectName extracts the project name from a git remote URL
func ExtractProjectName(remoteURL string) string {
	// Handle various URL formats:
	// git@github.com:user/repo.git
	// https://github.com/user/repo.git
	// https://github.com/user/repo

	// Remove .git suffix
	url := strings.TrimSuffix(remoteURL, ".git")

	// Try SSH format first: git@host:user/repo
	sshRegex := regexp.MustCompile(`git@[^:]+:(.+)`)
	if matches := sshRegex.FindStringSubmatch(url); len(matches) > 1 {
		parts := strings.Split(matches[1], "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	// Try HTTPS format: https://host/user/repo
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return ""
}

// GetCommits returns recent commits
func GetCommits(since string, limit int) ([]Commit, error) {
	args := []string{
		"log",
		"--oneline",
		"--no-decorate",
		fmt.Sprintf("--max-count=%d", limit),
	}

	if since != "" {
		args = append(args, fmt.Sprintf("--since=%s", since))
	}

	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}

	var commits []Commit
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) >= 2 {
			commits = append(commits, Commit{
				Hash:    parts[0],
				Message: parts[1],
			})
		}
	}

	return commits, nil
}

// GetCommitsDetailed returns detailed commit information
func GetCommitsDetailed(since string, limit int) ([]Commit, error) {
	args := []string{
		"log",
		"--format=%H|%s|%an|%ad",
		"--date=short",
		fmt.Sprintf("--max-count=%d", limit),
	}

	if since != "" {
		args = append(args, fmt.Sprintf("--since=%s", since))
	}

	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}

	var commits []Commit
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) >= 4 {
			commits = append(commits, Commit{
				Hash:    parts[0][:7], // Short hash
				Message: parts[1],
				Author:  parts[2],
				Date:    parts[3],
			})
		}
	}

	return commits, nil
}

// GetTodayCommits returns commits from today
func GetTodayCommits() ([]Commit, error) {
	today := time.Now().Format("2006-01-02")
	return GetCommits(today, 50)
}

// GetContext returns the full git context for the current directory
func GetContext() (*Context, error) {
	if !IsInGitRepo() {
		return nil, fmt.Errorf("not in a git repository")
	}

	ctx := &Context{}

	// Get remote URL
	if remote, err := GetRemoteURL(); err == nil {
		ctx.Remote = remote
		ctx.Project = ExtractProjectName(remote)
	}

	// Get current branch
	if branch, err := GetCurrentBranch(); err == nil {
		ctx.Branch = branch
	}

	return ctx, nil
}

// GetContextWithCommits returns git context including recent commits
func GetContextWithCommits(since string, limit int) (*Context, error) {
	ctx, err := GetContext()
	if err != nil {
		return nil, err
	}

	// Get commits
	if commits, err := GetCommits(since, limit); err == nil {
		ctx.Commits = commits
	}

	return ctx, nil
}

// FormatCommitsMarkdown formats commits as markdown
func FormatCommitsMarkdown(commits []Commit, includeHashes bool) string {
	var buf bytes.Buffer

	for _, c := range commits {
		if includeHashes {
			buf.WriteString(fmt.Sprintf("- `%s` %s\n", c.Hash, c.Message))
		} else {
			buf.WriteString(fmt.Sprintf("- %s\n", c.Message))
		}
	}

	return buf.String()
}

// GetDiff returns the diff for the working directory
func GetDiff(staged bool) (string, error) {
	args := []string{"diff"}
	if staged {
		args = append(args, "--staged")
	}

	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}

	return string(out), nil
}

// GetStatus returns the git status
func GetStatus() (string, error) {
	cmd := exec.Command("git", "status", "--short")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}
