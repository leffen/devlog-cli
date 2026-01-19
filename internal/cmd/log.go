package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/leffen/devlog-cli/internal/api"
	"github.com/leffen/devlog-cli/internal/config"
	"github.com/leffen/devlog-cli/internal/git"
	"github.com/spf13/cobra"
)

var (
	logSince      string
	logFormat     string
	logDryRun     bool
	logProject    string
	logVisibility string
	logLimit      int
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Create an entry from git commits",
	Long: `Create a DevLog entry from recent git commits.

This command reads your git log and creates an entry with a formatted
list of commits. It's great for summarizing what you've done.

Examples:
  # Create entry from today's commits
  devlog log

  # Commits from the last 3 days
  devlog log --since="3 days ago"

  # Commits from last week
  devlog log --since="1 week ago"

  # Preview without creating (dry run)
  devlog log --dry-run

  # Use summary format (grouped)
  devlog log --format=summary`,
	Run: runLog,
}

func runLog(cmd *cobra.Command, args []string) {
	// Check if in git repo
	if !git.IsInGitRepo() {
		fmt.Fprintln(os.Stderr, "Error: Not in a git repository")
		os.Exit(1)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
		os.Exit(1)
	}

	if cfg.APIKey == "" {
		fmt.Fprintln(os.Stderr, "Error: API key not configured. Run 'devlog config init' first.")
		os.Exit(1)
	}

	// Use defaults from config if not specified
	if logProject == "" {
		logProject = cfg.Defaults.Project
	}
	if logVisibility == "" {
		logVisibility = cfg.Defaults.Visibility
	}

	// Default to today's commits
	since := logSince
	if since == "" {
		since = time.Now().Format("2006-01-02")
	}

	// Get git context
	gitCtx, err := git.GetContextWithCommits(since, logLimit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting git context: %s\n", err)
		os.Exit(1)
	}

	if len(gitCtx.Commits) == 0 {
		fmt.Printf("No commits found since %s\n", since)
		os.Exit(0)
	}

	// Generate title
	title := generateLogTitle(gitCtx, since)

	// Generate content based on format
	var content string
	switch logFormat {
	case "summary":
		content = generateSummaryContent(gitCtx)
	default:
		content = generateListContent(gitCtx)
	}

	// Dry run - just print what would be created
	if logDryRun {
		fmt.Println("=== DRY RUN ===")
		fmt.Println()
		fmt.Printf("Title: %s\n", title)
		fmt.Printf("Project: %s\n", logProject)
		fmt.Printf("Commits: %d\n", len(gitCtx.Commits))
		fmt.Println()
		fmt.Println("Content:")
		fmt.Println("--------")
		fmt.Println(content)
		return
	}

	// Build request
	req := &api.CreateEntryRequest{
		Title:      title,
		Content:    content,
		Context:    logProject,
		Visibility: logVisibility,
		Git:        api.BuildGitInfo(gitCtx),
	}

	// Create API client
	client, err := api.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	// Create entry
	resp, err := client.CreateEntry(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating entry: %s\n", err)
		os.Exit(1)
	}

	// Output success
	fmt.Printf("✓ Entry created from %d commits\n", len(gitCtx.Commits))
	fmt.Printf("  Title: %s\n", resp.Title)
	fmt.Printf("  ID: %s\n", resp.ID)
}

func generateLogTitle(ctx *git.Context, since string) string {
	projectName := ctx.Project
	if projectName == "" {
		projectName = "Repository"
	}

	commitCount := len(ctx.Commits)
	today := time.Now().Format("2006-01-02")

	if since == today || since == "today" {
		return fmt.Sprintf("Git log: %s (%d commits today)", projectName, commitCount)
	}

	return fmt.Sprintf("Git log: %s (%d commits since %s)", projectName, commitCount, since)
}

func generateListContent(ctx *git.Context) string {
	var sb strings.Builder

	// Add context header
	if ctx.Branch != "" {
		sb.WriteString(fmt.Sprintf("Branch: `%s`\n\n", ctx.Branch))
	}

	sb.WriteString("## Commits\n\n")

	for _, c := range ctx.Commits {
		sb.WriteString(fmt.Sprintf("- `%s` %s\n", c.Hash, c.Message))
	}

	return sb.String()
}

func generateSummaryContent(ctx *git.Context) string {
	var sb strings.Builder

	// Add context header
	if ctx.Branch != "" {
		sb.WriteString(fmt.Sprintf("Branch: `%s`\n\n", ctx.Branch))
	}

	// Group commits by type (based on conventional commit prefixes)
	groups := make(map[string][]git.Commit)
	var other []git.Commit

	for _, c := range ctx.Commits {
		msg := c.Message
		categorized := false

		// Check for conventional commit prefixes
		prefixes := []struct {
			patterns []string
			label    string
		}{
			{[]string{"feat:", "feat(", "feature:"}, "Features"},
			{[]string{"fix:", "fix(", "bugfix:"}, "Bug Fixes"},
			{[]string{"docs:", "doc:"}, "Documentation"},
			{[]string{"test:", "tests:"}, "Tests"},
			{[]string{"refactor:", "refactor("}, "Refactoring"},
			{[]string{"chore:", "chore("}, "Chores"},
			{[]string{"style:", "style("}, "Style"},
			{[]string{"perf:", "perf("}, "Performance"},
		}

		for _, p := range prefixes {
			for _, pattern := range p.patterns {
				if strings.HasPrefix(strings.ToLower(msg), pattern) {
					groups[p.label] = append(groups[p.label], c)
					categorized = true
					break
				}
			}
			if categorized {
				break
			}
		}

		if !categorized {
			other = append(other, c)
		}
	}

	// Write grouped commits
	writeGroup := func(label string, commits []git.Commit) {
		if len(commits) == 0 {
			return
		}
		sb.WriteString(fmt.Sprintf("## %s\n\n", label))
		for _, c := range commits {
			// Remove the prefix from the message for cleaner display
			msg := c.Message
			for _, prefix := range []string{"feat:", "fix:", "docs:", "test:", "refactor:", "chore:", "style:", "perf:", "feature:", "bugfix:"} {
				if strings.HasPrefix(strings.ToLower(msg), prefix) {
					msg = strings.TrimSpace(msg[len(prefix):])
					break
				}
			}
			sb.WriteString(fmt.Sprintf("- %s\n", msg))
		}
		sb.WriteString("\n")
	}

	// Write in preferred order
	order := []string{"Features", "Bug Fixes", "Performance", "Refactoring", "Documentation", "Tests", "Style", "Chores"}
	for _, label := range order {
		writeGroup(label, groups[label])
	}

	// Write uncategorized commits
	if len(other) > 0 {
		sb.WriteString("## Other Changes\n\n")
		for _, c := range other {
			sb.WriteString(fmt.Sprintf("- `%s` %s\n", c.Hash, c.Message))
		}
	}

	return sb.String()
}

func init() {
	rootCmd.AddCommand(logCmd)

	logCmd.Flags().StringVar(&logSince, "since", "", "Include commits since this date (default: today)")
	logCmd.Flags().StringVar(&logFormat, "format", "list", "Output format: list or summary")
	logCmd.Flags().BoolVar(&logDryRun, "dry-run", false, "Preview without creating entry")
	logCmd.Flags().StringVarP(&logProject, "project", "p", "", "Project context (default from config)")
	logCmd.Flags().StringVarP(&logVisibility, "visibility", "v", "", "Visibility (default from config)")
	logCmd.Flags().IntVar(&logLimit, "limit", 50, "Maximum number of commits to include")
}
