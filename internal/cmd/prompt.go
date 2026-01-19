package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var promptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Output Claude prompt for CLAUDE.md",
	Long: `Output a prompt/instructions for Claude Code that can be appended to CLAUDE.md.

Example:
  devlog prompt >> CLAUDE.md`,
	Run: runPrompt,
}

func runPrompt(cmd *cobra.Command, args []string) {
	fmt.Print(claudePrompt)
}

const claudePrompt = `
## DevLog CLI Integration

This project uses DevLog CLI to log development progress. Use it to create entries when completing significant work.

### Quick Commands

` + "```" + `bash
# Log a completed task
devlog new -t "Title" -c "Description of what was done"

# Log with git context (includes branch, commits)
devlog new -t "Feature complete" -c "Description" --include-git

# Create entry from today's commits
devlog log

# Create entry from stdin (useful for piping)
echo "Implemented feature X" | devlog new -t "Feature X" -

# Session summary
devlog summary --session "Feature name" --achievements "What was done" --next-steps "What's next"
` + "```" + `

### When to Log

- After completing a feature or significant change
- After fixing a bug
- At the end of a coding session
- When making architectural decisions

### Entry Guidelines

- Keep titles concise and descriptive
- Use --include-git when the commit history is relevant
- Add tags for categorization: --tags "feature,backend"
- Use context flags: job (work), project (personal), fun (experimental)
`

func init() {
	rootCmd.AddCommand(promptCmd)
}
