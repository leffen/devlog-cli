package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version = "dev"
	// Commit is set at build time
	Commit = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "devlog",
	Short: "DevLog Daily CLI - Journal your development progress",
	Long: `DevLog is a command-line tool for creating entries in DevLog Daily.

It automatically detects git context and makes it easy to log your
development progress from the terminal or integrate with tools like
Claude Code.

Examples:
  # Create a simple entry
  devlog new -t "Fixed auth bug" -c "Updated the login flow"

  # Create an entry from stdin (great for Claude Code)
  echo "Implemented OAuth2" | devlog new -t "Auth Implementation" -

  # Create an entry with git context
  devlog new -t "Feature complete" --include-git

  # Create entry from today's commits
  devlog log

  # List recent entries
  devlog list

  # Configure the CLI
  devlog config set api-key YOUR_API_KEY`,
	Version: fmt.Sprintf("%s (commit: %s)", Version, Commit),
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug output")
}
