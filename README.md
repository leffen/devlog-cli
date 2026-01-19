# DevLog CLI

A command-line tool for creating entries in DevLog Daily. Perfect for developers who want to log their progress from the terminal or integrate with tools like Claude Code.

## Features

- **Auto-detect git context** - Remote URL, project name, current branch
- **Multiple input modes** - Stdin, file, editor, flags
- **Git log integration** - Create entries from today's commits
- **Session summaries** - Structured summary of work sessions
- **Claude Code integration** - Easy to use with AI coding assistants

## Installation

### Homebrew (macOS/Linux)

```bash
brew install leffen/tap/devlog
```

### Go Install

```bash
go install github.com/leffen/devlog-cli/cmd/devlog@latest
```

### Download Binary

Download the latest release from the [GitHub Releases](https://github.com/leffen/devlog-cli/releases) page.

### From source

```bash
git clone https://github.com/leffen/devlog-cli.git
cd devlog-cli
make build
make install-local  # Installs to /usr/local/bin
```

## Configuration

### Interactive setup

```bash
devlog config init
```

### Manual configuration

```bash
devlog config set api-key YOUR_API_KEY
devlog config set server https://devlog.asd09.com
devlog config set project job  # default project context
```

### Configuration file

Configuration is stored in `~/.config/devlog/config.yaml`:

```yaml
api_key: dlk_your_api_key_here
server: https://devlog.asd09.com
defaults:
  project: job        # job | project | fun
  visibility: private
  include_git: true
```

You can also create a `.devlog.yaml` in your project directory for project-specific settings.

## Usage

### Create a new entry

```bash
# Simple entry
devlog new -t "Fixed auth bug" -c "Updated login flow"

# From stdin (great for Claude Code)
echo "Implemented OAuth2 login flow" | devlog new -t "Auth Implementation" -

# From a file
devlog new -t "Session Notes" -f notes.md

# With git context
devlog new -t "Feature complete" --include-git

# Open editor
devlog new --editor --auto-title

# With tags
devlog new -t "Bug fix" -c "Fixed memory leak" --tags "bugfix,performance"
```

### Create entry from git commits

```bash
# Today's commits
devlog log

# Commits from last week
devlog log --since="1 week ago"

# Preview without creating
devlog log --dry-run

# Summary format (groups by commit type)
devlog log --format=summary
```

### Create a session summary

```bash
devlog summary \
  --session "Authentication system" \
  --time "3h" \
  --achievements "OAuth2 complete, tests passing" \
  --blockers "Need review for PR" \
  --next-steps "Add password reset"
```

### List recent entries

```bash
# Last 10 entries
devlog list

# More entries
devlog list --limit 20

# Filter by project
devlog list --project job

# Filter by date
devlog list --since "1 week ago"

# Compact format
devlog list --format compact
```

## Claude Code Integration

The CLI is designed to work seamlessly with Claude Code:

```bash
# Quick feature note
echo "Implemented OAuth2 login flow" | devlog new -t "Auth Implementation" -

# Session summary
devlog summary \
  --session "Authentication system" \
  --time "3h" \
  --achievements "OAuth2 complete, tests passing" \
  --next-steps "Add password reset"

# Git log for today
devlog log --since="today"

# Full entry with git context
devlog new -t "Feature: Search" --include-git --tags "search,backend" - <<'EOF'
## Implementation
- Added Elasticsearch integration
- Implemented fuzzy search

## Files Changed
- src/search/index.ts
EOF
```

## API Key Setup

1. Log in to DevLog Daily at https://devlog.asd09.com
2. Go to Settings → API Keys
3. Click "Create API Key"
4. Copy the key and save it securely
5. Configure the CLI:

```bash
devlog config set api-key YOUR_API_KEY
```

## Development

### Building

```bash
make build
```

### Running tests

```bash
make test
```

### Building for all platforms

```bash
make build-all
```

### Creating a Release

Releases are managed with [GoReleaser](https://goreleaser.com/).

```bash
# Test release locally (no publish)
make snapshot

# Dry run (validates everything)
make release-dry

# Create and publish release (requires GITHUB_TOKEN)
make release
```

To create a new release:

1. Tag the commit:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. GitHub Actions will automatically build and publish the release.
