package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/leffen/devlog-cli/internal/punch"
	"github.com/spf13/cobra"
)

var (
	punchProject string
	punchComment string

	punchListLimit   int
	punchListProject string
	punchListAll     bool
)

var punchCmd = &cobra.Command{
	Use:   "punch [comment]",
	Short: "Timestamp work on a project",
	Long: `Record a timestamped work punch for the current project.

Each punch captures the time, machine name, local IP address, current
directory, and a sanitized project name derived from the directory name. An
optional comment can be attached. Records are stored locally as JSONL in the
config directory.

Examples:
  # Punch in for the current project (project derived from directory name)
  devlog punch

  # Punch with a comment
  devlog punch "started on the auth refactor"
  devlog punch -m "started on the auth refactor"

  # Override the project name
  devlog punch -p backend "deploying"

  # View a compact history
  devlog punch list`,
	Args: cobra.ArbitraryArgs,
	Run:  runPunch,
}

var punchListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show a compact history of work punches",
	Long: `Show a compact, chronological history of work punches.

By default only punches for the current directory's project are shown. Use
--all to show every project, or --project to filter by a specific one.

Examples:
  # Recent punches for the current project
  devlog punch list

  # All projects
  devlog punch list --all

  # A specific project
  devlog punch list --project backend

  # Show more rows
  devlog punch list --limit 50`,
	Run: runPunchList,
}

func runPunch(cmd *cobra.Command, args []string) {
	// A comment may be given positionally or via -m/--comment.
	comment := punchComment
	if comment == "" && len(args) > 0 {
		comment = strings.Join(args, " ")
	}

	rec, err := punch.NewRecord(punchProject, comment)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	if err := punch.Append(rec); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving punch: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Punched %s @ %s\n", rec.Project, rec.Timestamp.Format("Jan 02 15:04"))
	fmt.Printf("  Machine: %s", rec.Machine)
	if rec.IP != "" {
		fmt.Printf(" (%s)", rec.IP)
	}
	fmt.Println()
	fmt.Printf("  Dir:     %s\n", rec.Dir)
	if rec.Comment != "" {
		fmt.Printf("  Comment: %s\n", rec.Comment)
	}
}

func runPunchList(cmd *cobra.Command, args []string) {
	records, err := punch.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading punches: %s\n", err)
		os.Exit(1)
	}

	// Determine the project filter.
	filter := punch.SanitizeProject(punchListProject)
	if filter == "" && !punchListAll {
		if dir, err := os.Getwd(); err == nil {
			filter = punch.ProjectFromDir(dir)
		}
	}

	if filter != "" {
		filtered := records[:0:0]
		for _, r := range records {
			if r.Project == filter {
				filtered = append(filtered, r)
			}
		}
		records = filtered
	}

	if len(records) == 0 {
		if filter != "" {
			fmt.Printf("No punches found for project '%s' (use --all to show every project)\n", filter)
		} else {
			fmt.Println("No punches found")
		}
		return
	}

	// Most recent first.
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].Timestamp.After(records[j].Timestamp)
	})

	if punchListLimit > 0 && len(records) > punchListLimit {
		records = records[:punchListLimit]
	}

	for _, r := range records {
		when := r.Timestamp.Format("2006-01-02 15:04")
		line := fmt.Sprintf("%s  %-16s  %s", when, truncate(r.Project, 16), r.Machine)
		if r.Comment != "" {
			line += "  " + r.Comment
		}
		fmt.Println(line)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}

func init() {
	rootCmd.AddCommand(punchCmd)
	punchCmd.AddCommand(punchListCmd)

	punchCmd.Flags().StringVarP(&punchProject, "project", "p", "", "Project name (default: sanitized current directory name)")
	punchCmd.Flags().StringVarP(&punchComment, "comment", "m", "", "Optional comment")

	punchListCmd.Flags().IntVar(&punchListLimit, "limit", 20, "Maximum number of punches to show (0 = all)")
	punchListCmd.Flags().StringVarP(&punchListProject, "project", "p", "", "Filter by project (default: current directory's project)")
	punchListCmd.Flags().BoolVar(&punchListAll, "all", false, "Show punches for all projects")
}
