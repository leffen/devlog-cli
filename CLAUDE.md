# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

DevLog CLI is a command-line tool for creating entries in DevLog Daily (https://devlog.asd09.com). It's built with Go and uses the Cobra CLI framework. The tool auto-detects git context, supports multiple input modes (stdin, file, editor), and integrates seamlessly with AI coding assistants like Claude Code.

## Commands for Development

### Build & Install
```bash
# Build the binary
make build

# Install to GOPATH/bin
make install

# Install to /usr/local/bin (requires sudo)
make install-local

# Build for all platforms (darwin, linux, windows)
make build-all
```

### Testing
```bash
# Run all tests
make test

# Run tests with verbose output
go test -v ./...

# Run tests for specific package
go test -v ./internal/config
```

### Code Quality
```bash
# Format code
make fmt
go fmt ./...

# Lint code (requires golangci-lint)
make lint

# Vet code
go vet ./...

# Update dependencies
make deps
```

### Release Management
```bash
# Test release locally (no publish)
make snapshot

# Dry run release (validates everything)
make release-dry

# Check goreleaser config
make check

# Create and publish release (requires GITHUB_TOKEN)
make release
```

### Local Testing
```bash
# Run directly with go run
go run ./main.go --help
go run ./main.go new -t "Test" -c "Content"

# After building
./devlog --help
./devlog new -t "Test" -c "Content"
```

## Architecture & Code Organization

### Entry Point & CLI Structure
- `main.go`: Minimal entry point that delegates to `cmd.Execute()`
- `cmd/root.go`: Root Cobra command with version info and global flags
- `cmd/*.go`: Each subcommand (new, log, list, summary, config) in separate files

### Core Packages

**internal/config**: Configuration management
- `config.Config`: Main config struct with API key, server URL, and defaults
- Two-tier config system: global (`~/.config/devlog/config.yaml`) and project-local (`.devlog.yaml`)
- Project-local config overrides global settings
- Defaults: server (https://devlog.asd09.com), project (job), visibility (private), include_git (true)

**internal/api**: DevLog API client
- `api.Client`: HTTP client with 30s timeout, bearer auth via X-API-Key header
- `CreateEntry()`: POST to /api/cli/entries
- `ListEntries()`: GET from /api/cli/entries with query params
- All requests use JSON content type and include User-Agent: DevLog-CLI/1.0

**internal/git**: Git integration utilities
- `git.Context`: Struct containing remote URL, project name, branch, and commits
- `IsInGitRepo()`: Check if current directory is in a git repo
- `GetContext()`: Retrieves remote, project name, and branch
- `GetCommits()`: Fetches commit history with --since and --max-count
- `GetCommitsDetailed()`: Returns commits with author and date info
- `ExtractProjectName()`: Parses repo name from git remote URL (handles SSH and HTTPS formats)
- `FormatCommitsMarkdown()`: Generates markdown list from commits

### Key Subcommands

**new**: Create new entry
- Multiple input modes: flags (-c), stdin (-), file (-f), editor (--editor)
- Auto-title generation from content first line or git branch
- Git context inclusion via --include-git flag (uses config default)
- Editor support: respects $EDITOR or $VISUAL, falls back to vi
- Template in editor includes comment lines that are stripped on save

**log**: Create entry from git commits
- Defaults to today's commits if --since not specified
- Two formats: "list" (default) and "summary"
- Summary format groups commits by conventional commit prefixes (feat:, fix:, docs:, etc.)
- Dry-run mode to preview without creating entry
- Automatically includes git context in API request

**list**: List recent entries
- Supports --limit, --since, and --project filters
- Format options: default and compact

**summary**: Create structured session summary
- Specific flags for session, time, achievements, blockers, next-steps
- Generates markdown-formatted entry with sections

**config**: Manage configuration
- `config init`: Interactive setup wizard
- `config set <key> <value>`: Set individual values
- `config get <key>`: Retrieve configuration values
- Supports keys: api-key, server, project, visibility, include-git

### Version Information
Build-time variables set via ldflags:
- `cmd.Version`: Git tag or "dev"
- `cmd.Commit`: Short commit hash
- Display: `devlog --version` shows both

## Development Patterns

### Adding New Subcommands
1. Create new file in `cmd/` (e.g., `cmd/mycommand.go`)
2. Define command struct using Cobra pattern
3. Implement `run<CommandName>` function
4. Register command in `init()` with `rootCmd.AddCommand(myCmd)`
5. Add flags using `myCmd.Flags()`

### Working with Config
Always use two-step config loading:
```go
cfg, err := config.Load()  // Loads global + project-local merged
// Use cfg.Defaults for fallback values when flags not set
// Check cmd.Flags().Changed("flag-name") to distinguish unset vs false
```

### Git Integration Best Practices
- Always check `git.IsInGitRepo()` before calling git functions
- Handle errors gracefully - git context is optional for most commands
- Use `GetContextWithCommits()` when you need both metadata and commit history
- Respect the --since flag for commit queries (default to today)

### API Error Handling
- Check for non-2xx status codes
- Parse ErrorResponse JSON structure: `{"error": "message"}`
- Provide user-friendly error messages
- Exit with status 1 on errors

### Release Workflow
1. GoReleaser builds for multiple platforms (linux, darwin, windows; amd64, arm64)
2. Archives created in both tar.gz and zip formats
3. Releases tagged with `cli/v*` pattern (e.g., `cli/v1.0.0`)
4. GitHub Actions automatically builds and publishes on tag push
5. Version and commit injected at build time via ldflags

## Configuration System

### Config Hierarchy
1. Global config: `~/.config/devlog/config.yaml` (or `$XDG_CONFIG_HOME/devlog/config.yaml`)
2. Project-local config: `.devlog.yaml` in current directory
3. Command-line flags (highest priority)

### Project Context Values
- `job`: Work-related development
- `project`: Personal projects
- `fun`: Experimental/hobby coding

### Visibility Values
- `private`: Default, entry visible only to user
- `public`: Entry can be shared

## API Integration Notes

### Authentication
- API key obtained from DevLog Daily web UI (Settings → API Keys)
- Passed via `X-API-Key` header on all requests
- No OAuth or token refresh - static API keys

### Entry Structure
An entry consists of:
- **title** (required): Entry headline
- **content** (required): Main entry body (supports markdown)
- **context** (optional): Project context (job/project/fun)
- **mood** (optional): Emoji or text mood indicator
- **tags** (optional): Array of tag strings
- **visibility** (optional): private or public
- **git** (optional): Git context object with remote, project, branch, commits

### Git Context Format
```go
{
  "remote": "git@github.com:user/repo.git",
  "project": "repo",
  "branch": "main",
  "commits": [
    {"hash": "abc1234", "message": "feat: add feature"},
    {"hash": "def5678", "message": "fix: resolve bug"}
  ]
}
```

## Testing Considerations

When writing tests:
- Mock API calls - don't hit the real DevLog API
- Test config loading with temporary directories
- Test git functions require being in a git repo (use test fixtures or skip)
- Test conventional commit parsing in summary format
- Verify editor cleanup of comment lines
