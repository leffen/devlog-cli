package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/leffen/devlog-cli/internal/api"
	"github.com/leffen/devlog-cli/internal/config"
	"github.com/spf13/cobra"
)

var (
	listLimit   int
	listSince   string
	listProject string
	listFormat  string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent DevLog entries",
	Long: `List your recent DevLog entries.

Examples:
  # List last 10 entries (default)
  devlog list

  # List more entries
  devlog list --limit 20

  # Filter by project
  devlog list --project job

  # Filter by date
  devlog list --since "1 week ago"

  # Compact format
  devlog list --format compact`,
	Run: runList,
}

func runList(cmd *cobra.Command, args []string) {
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

	// Create API client
	client, err := api.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	// Parse since date
	since := ""
	if listSince != "" {
		// Try to parse relative dates
		since = parseRelativeDate(listSince)
	}

	// Fetch entries
	entries, err := client.ListEntries(listLimit, since, listProject)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching entries: %s\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Println("No entries found")
		return
	}

	// Display based on format
	switch listFormat {
	case "compact":
		displayCompact(entries)
	case "json":
		displayJSON(entries)
	default:
		displayDefault(entries)
	}
}

func parseRelativeDate(s string) string {
	// Handle common relative date formats
	now := time.Now()
	s = strings.ToLower(strings.TrimSpace(s))

	// Map of relative terms
	if s == "today" {
		return now.Format("2006-01-02")
	}
	if s == "yesterday" {
		return now.AddDate(0, 0, -1).Format("2006-01-02")
	}
	if strings.Contains(s, "week ago") || strings.Contains(s, "weeks ago") {
		weeks := 1
		fmt.Sscanf(s, "%d week", &weeks)
		if weeks == 0 {
			fmt.Sscanf(s, "%d weeks", &weeks)
		}
		if weeks == 0 {
			weeks = 1
		}
		return now.AddDate(0, 0, -7*weeks).Format("2006-01-02")
	}
	if strings.Contains(s, "day ago") || strings.Contains(s, "days ago") {
		days := 1
		fmt.Sscanf(s, "%d day", &days)
		if days == 0 {
			fmt.Sscanf(s, "%d days", &days)
		}
		if days == 0 {
			days = 1
		}
		return now.AddDate(0, 0, -days).Format("2006-01-02")
	}
	if strings.Contains(s, "month ago") || strings.Contains(s, "months ago") {
		months := 1
		fmt.Sscanf(s, "%d month", &months)
		if months == 0 {
			fmt.Sscanf(s, "%d months", &months)
		}
		if months == 0 {
			months = 1
		}
		return now.AddDate(0, -months, 0).Format("2006-01-02")
	}

	// Try to parse as ISO date
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.Format("2006-01-02")
	}

	// Return as-is and let the server handle it
	return s
}

func displayDefault(entries []api.Entry) {
	fmt.Printf("Found %d entries\n\n", len(entries))

	for i, entry := range entries {
		// Parse date
		createdAt, _ := time.Parse(time.RFC3339, entry.CreatedAt)
		dateStr := createdAt.Format("Jan 2, 2006 3:04 PM")

		// Header
		fmt.Printf("═══════════════════════════════════════════════════════\n")
		fmt.Printf("  %s\n", entry.Title)
		fmt.Printf("───────────────────────────────────────────────────────\n")

		// Metadata
		fmt.Printf("  Date:    %s\n", dateStr)
		fmt.Printf("  Context: %s\n", entry.Context)
		if len(entry.Tags) > 0 {
			fmt.Printf("  Tags:    %s\n", strings.Join(entry.Tags, ", "))
		}
		if entry.Mood != "" {
			fmt.Printf("  Mood:    %s\n", entry.Mood)
		}
		fmt.Printf("  ID:      %s\n", entry.ID)

		// Content preview
		preview := getContentPreview(entry.Content, 200)
		if preview != "" {
			fmt.Println()
			fmt.Printf("  %s\n", preview)
		}

		if i < len(entries)-1 {
			fmt.Println()
		}
	}
	fmt.Printf("═══════════════════════════════════════════════════════\n")
}

func displayCompact(entries []api.Entry) {
	for _, entry := range entries {
		createdAt, _ := time.Parse(time.RFC3339, entry.CreatedAt)
		dateStr := createdAt.Format("Jan 02")

		// Truncate title if needed
		title := entry.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}

		fmt.Printf("[%s] %-8s %s\n", dateStr, entry.Context, title)
	}
}

func displayJSON(entries []api.Entry) {
	// Simple JSON output
	fmt.Println("[")
	for i, entry := range entries {
		comma := ","
		if i == len(entries)-1 {
			comma = ""
		}
		fmt.Printf(`  {"id":"%s","title":"%s","context":"%s","createdAt":"%s"}%s`+"\n",
			entry.ID, escapeJSON(entry.Title), entry.Context, entry.CreatedAt, comma)
	}
	fmt.Println("]")
}

func getContentPreview(content string, maxLen int) string {
	// Remove markdown headers
	lines := strings.Split(content, "\n")
	var preview strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip headers
		if strings.HasPrefix(line, "#") {
			continue
		}
		// Skip markdown links/images
		if strings.HasPrefix(line, "![") || strings.HasPrefix(line, "[") {
			continue
		}

		if preview.Len() > 0 {
			preview.WriteString(" ")
		}
		preview.WriteString(line)

		if preview.Len() >= maxLen {
			break
		}
	}

	result := preview.String()
	if len(result) > maxLen {
		result = result[:maxLen-3] + "..."
	}
	return result
}

func escapeJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().IntVar(&listLimit, "limit", 10, "Maximum number of entries to show")
	listCmd.Flags().StringVar(&listSince, "since", "", "Show entries since this date (e.g., 'today', '1 week ago')")
	listCmd.Flags().StringVarP(&listProject, "project", "p", "", "Filter by project (job, project, fun)")
	listCmd.Flags().StringVarP(&listFormat, "format", "f", "default", "Output format: default, compact, json")
}
