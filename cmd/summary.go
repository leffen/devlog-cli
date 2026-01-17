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
	summarySession      string
	summaryTime         string
	summaryAchievements string
	summaryBlockers     string
	summaryNextSteps    string
	summaryProject      string
	summaryVisibility   string
	summaryMood         string
	summaryTags         []string
	summaryIncludeGit   bool
)

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Create a structured session summary",
	Long: `Create a structured session summary entry.

This command creates an entry with a standardized format for summarizing
a work session. It's perfect for end-of-day recaps or session handoffs.

Examples:
  # Full session summary
  devlog summary \
    --session "Authentication system" \
    --time "3h" \
    --achievements "OAuth2 complete, tests passing" \
    --blockers "Need review for PR" \
    --next-steps "Add password reset"

  # Quick summary
  devlog summary --session "Bug fixes" --achievements "Fixed 3 critical bugs"

  # With git context
  devlog summary --session "Feature work" --include-git`,
	Run: runSummary,
}

func runSummary(cmd *cobra.Command, args []string) {
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

	// Validate required fields
	if summarySession == "" {
		fmt.Fprintln(os.Stderr, "Error: --session is required")
		os.Exit(1)
	}

	if summaryAchievements == "" {
		fmt.Fprintln(os.Stderr, "Error: --achievements is required")
		os.Exit(1)
	}

	// Use defaults from config if not specified
	if summaryProject == "" {
		summaryProject = cfg.Defaults.Project
	}
	if summaryVisibility == "" {
		summaryVisibility = cfg.Defaults.Visibility
	}
	if !cmd.Flags().Changed("include-git") {
		summaryIncludeGit = cfg.Defaults.IncludeGit
	}

	// Generate title
	title := fmt.Sprintf("Session: %s", summarySession)
	if summaryTime != "" {
		title = fmt.Sprintf("Session: %s (%s)", summarySession, summaryTime)
	}

	// Generate content
	content := generateSummaryStructure()

	// Build request
	req := &api.CreateEntryRequest{
		Title:      title,
		Content:    content,
		Context:    summaryProject,
		Mood:       summaryMood,
		Tags:       summaryTags,
		Visibility: summaryVisibility,
	}

	// Add git context if requested
	if summaryIncludeGit && git.IsInGitRepo() {
		gitCtx, err := git.GetContext()
		if err == nil {
			req.Git = api.BuildGitInfo(gitCtx)
		}
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
	fmt.Printf("✓ Session summary created: %s\n", resp.Title)
	fmt.Printf("  ID: %s\n", resp.ID)
}

func generateSummaryStructure() string {
	var sb strings.Builder

	// Session header
	sb.WriteString("## Session Summary\n\n")

	// Time spent
	if summaryTime != "" {
		sb.WriteString(fmt.Sprintf("**Time spent:** %s\n\n", summaryTime))
	}

	// Date
	sb.WriteString(fmt.Sprintf("**Date:** %s\n\n", time.Now().Format("January 2, 2006")))

	// Achievements
	sb.WriteString("## Achievements\n\n")
	for _, item := range splitItems(summaryAchievements) {
		sb.WriteString(fmt.Sprintf("- %s\n", item))
	}
	sb.WriteString("\n")

	// Blockers (if provided)
	if summaryBlockers != "" {
		sb.WriteString("## Blockers\n\n")
		for _, item := range splitItems(summaryBlockers) {
			sb.WriteString(fmt.Sprintf("- %s\n", item))
		}
		sb.WriteString("\n")
	}

	// Next Steps (if provided)
	if summaryNextSteps != "" {
		sb.WriteString("## Next Steps\n\n")
		for _, item := range splitItems(summaryNextSteps) {
			sb.WriteString(fmt.Sprintf("- [ ] %s\n", item))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// splitItems splits a comma-separated string into items
func splitItems(s string) []string {
	var items []string
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func init() {
	rootCmd.AddCommand(summaryCmd)

	summaryCmd.Flags().StringVar(&summarySession, "session", "", "Session/focus area name (required)")
	summaryCmd.Flags().StringVar(&summaryTime, "time", "", "Time spent (e.g., '3h', '2h30m')")
	summaryCmd.Flags().StringVar(&summaryAchievements, "achievements", "", "What was accomplished (comma-separated)")
	summaryCmd.Flags().StringVar(&summaryBlockers, "blockers", "", "Blockers encountered (comma-separated)")
	summaryCmd.Flags().StringVar(&summaryNextSteps, "next-steps", "", "Next steps/TODOs (comma-separated)")
	summaryCmd.Flags().StringVarP(&summaryProject, "project", "p", "", "Project context (default from config)")
	summaryCmd.Flags().StringVarP(&summaryVisibility, "visibility", "v", "", "Visibility (default from config)")
	summaryCmd.Flags().StringVarP(&summaryMood, "mood", "m", "", "Mood emoji or text")
	summaryCmd.Flags().StringSliceVar(&summaryTags, "tags", nil, "Tags (comma-separated)")
	summaryCmd.Flags().BoolVar(&summaryIncludeGit, "include-git", false, "Include git context")

	summaryCmd.MarkFlagRequired("session")
	summaryCmd.MarkFlagRequired("achievements")
}
