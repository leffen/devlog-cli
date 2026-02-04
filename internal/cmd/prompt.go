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

### Commands

` + "```" + `bash
# Log a completed task (always include --source to identify yourself)
devlog new -t "Title" -c "Description of what was done" --source claude-code

# Log with git context (includes branch, commits) and tags
devlog new -t "Feature complete" -c "Description" --include-git --tags "feature,backend" --source claude-code

# Specify repository explicitly (auto-detected from git when omitted)
devlog new -t "Title" -c "Content" --repo my-repo --source claude-code

# Set work context: job (work), project (personal), fun (experimental)
devlog new -t "Title" -c "Content" -p job --source claude-code

# Create entry from today's git commits
devlog log --source claude-code

# Create entry from stdin (useful for piping)
echo "Implemented feature X" | devlog new -t "Feature X" --source claude-code -

# Session summary (--session and --achievements are required)
devlog summary --session "Feature name" --achievements "What was done" --next-steps "What's next" --source claude-code
` + "```" + `

### Flag Reference

| Flag | Commands | Description |
|------|----------|-------------|
| ` + "`" + `-t, --title` + "`" + ` | new | Entry title (required unless --auto-title) |
| ` + "`" + `-c, --content` + "`" + ` | new | Entry content |
| ` + "`" + `--source` + "`" + ` | new, log, summary | Source identifier (e.g. claude-code, cursor) |
| ` + "`" + `--repo` + "`" + ` | new, log, summary | Repository name (auto-detected from git) |
| ` + "`" + `-p, --project` + "`" + ` | new, log, summary | Work context: job, project, fun |
| ` + "`" + `--tags` + "`" + ` | new, summary | Comma-separated tags |
| ` + "`" + `--include-git` + "`" + ` | new, summary | Include git context (remote, branch, commits) |
| ` + "`" + `-m, --mood` + "`" + ` | new, summary | Mood emoji or text |
| ` + "`" + `-v, --visibility` + "`" + ` | new, log, summary | private (default) or public |
| ` + "`" + `--session` + "`" + ` | summary | Session name (required) |
| ` + "`" + `--achievements` + "`" + ` | summary | What was accomplished (required, comma-separated) |
| ` + "`" + `--next-steps` + "`" + ` | summary | Next steps (comma-separated) |
| ` + "`" + `--since` + "`" + ` | log | Include commits since date (default: today) |
| ` + "`" + `--dry-run` + "`" + ` | log | Preview without creating entry |

### When to Log

- After completing a feature or significant change
- After fixing a bug
- At the end of a coding session
- When making architectural decisions

### Guidelines

- Always include ` + "`" + `--source claude-code` + "`" + ` to identify entries created by this agent
- Keep titles concise and descriptive (under 60 chars)
- Use ` + "`" + `--tags` + "`" + ` to categorize: feature, bugfix, refactor, docs, test, chore
- Use ` + "`" + `--include-git` + "`" + ` when the commit history is relevant
- The ` + "`" + `--repo` + "`" + ` flag is auto-detected from git; only set it explicitly if outside a git repo
`

func init() {
	rootCmd.AddCommand(promptCmd)
}
