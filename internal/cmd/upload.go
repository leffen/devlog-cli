package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/leffen/devlog-cli/internal/api"
	"github.com/leffen/devlog-cli/internal/config"
	"github.com/leffen/devlog-cli/internal/git"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	uploadTitle      string
	uploadProject    string
	uploadMood       string
	uploadTags       []string
	uploadVisibility string
	uploadIncludeGit bool
	uploadSource     string
	uploadRepo       string
	uploadDryRun     bool
)

// frontmatter holds the optional YAML metadata block at the top of a markdown file.
type frontmatter struct {
	Title      string   `yaml:"title"`
	Project    string   `yaml:"project"`
	Context    string   `yaml:"context"`
	Mood       string   `yaml:"mood"`
	Tags       []string `yaml:"tags"`
	Visibility string   `yaml:"visibility"`
	Source     string   `yaml:"source"`
	Repository string   `yaml:"repository"`
	Repo       string   `yaml:"repo"`
}

var uploadCmd = &cobra.Command{
	Use:   "upload <file.md> [file2.md ...]",
	Short: "Upload one or more markdown files as DevLog entries",
	Long: `Upload one or more markdown files as DevLog entries.

Each file becomes its own entry. The title is resolved in this order:
  1. The -t/--title flag (only when uploading a single file)
  2. A "title" field in the file's YAML frontmatter
  3. The first level-1 heading (# Title) in the file
  4. The file name (without extension)

Files may begin with an optional YAML frontmatter block, which sets
per-file metadata and overrides the command-line flags for that file:

  ---
  title: My Entry
  project: job
  tags: [refactor, api]
  visibility: private
  mood: 🚀
  ---

  # Heading

  Body content...

Examples:
  # Upload a single file
  devlog upload notes.md

  # Upload several files at once
  devlog upload daily/*.md

  # Apply shared metadata to all uploaded files
  devlog upload -p job --tags "standup,notes" monday.md tuesday.md

  # Preview without creating entries
  devlog upload --dry-run notes.md`,
	Args: cobra.MinimumNArgs(1),
	Run:  runUpload,
}

func runUpload(cmd *cobra.Command, args []string) {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
		os.Exit(1)
	}

	// Check API key (skip in dry-run so previews work offline)
	if cfg.APIKey == "" && !uploadDryRun {
		fmt.Fprintln(os.Stderr, "Error: API key not configured. Run 'devlog config init' first.")
		os.Exit(1)
	}

	// --title only makes sense for a single file
	if uploadTitle != "" && len(args) > 1 {
		fmt.Fprintln(os.Stderr, "Error: --title can only be used when uploading a single file.")
		os.Exit(1)
	}

	// Determine include-git default (flag overrides config default)
	includeGit := uploadIncludeGit
	if !cmd.Flags().Changed("include-git") {
		includeGit = cfg.Defaults.IncludeGit
	}

	// Build a shared git context once if requested
	var gitInfo *api.GitInfo
	if includeGit && git.IsInGitRepo() {
		if gitCtx, err := git.GetContext(); err == nil {
			gitInfo = api.BuildGitInfo(gitCtx)
		}
	}

	// Create API client (not needed for dry-run)
	var client *api.Client
	if !uploadDryRun {
		client, err = api.NewClient(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
	}

	var failures int
	for _, path := range args {
		if err := uploadFile(cmd, client, cfg, gitInfo, path); err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: %s\n", path, err)
			failures++
		}
	}

	if failures > 0 {
		fmt.Fprintf(os.Stderr, "\n%d of %d file(s) failed.\n", failures, len(args))
		os.Exit(1)
	}
}

func uploadFile(cmd *cobra.Command, client *api.Client, cfg *config.Config, gitInfo *api.GitInfo, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	fm, body, err := parseFrontmatter(string(raw))
	if err != nil {
		return fmt.Errorf("parsing frontmatter: %w", err)
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return fmt.Errorf("file has no content")
	}

	// Resolve title: --title flag > frontmatter > first H1 > filename
	title := uploadTitle
	if title == "" {
		title = fm.Title
	}
	if title == "" {
		title = extractMarkdownTitle(body)
	}
	if title == "" {
		title = titleFromFilename(path)
	}

	// Resolve metadata: explicit flag > frontmatter > config default
	project := resolve(cmd, "project", uploadProject, firstNonEmpty(fm.Project, fm.Context), cfg.Defaults.Project)
	visibility := resolve(cmd, "visibility", uploadVisibility, fm.Visibility, cfg.Defaults.Visibility)
	mood := resolve(cmd, "mood", uploadMood, fm.Mood, "")
	source := resolve(cmd, "source", uploadSource, fm.Source, "cli")

	repo := resolve(cmd, "repo", uploadRepo, firstNonEmpty(fm.Repository, fm.Repo), "")
	if repo == "" && git.IsInGitRepo() {
		if gitCtx, err := git.GetContext(); err == nil {
			repo = gitCtx.Project
		}
	}

	tags := uploadTags
	if len(tags) == 0 {
		tags = fm.Tags
	}

	req := &api.CreateEntryRequest{
		Title:      title,
		Content:    body,
		Context:    project,
		Mood:       mood,
		Tags:       tags,
		Visibility: visibility,
		Source:     source,
		Repository: repo,
		Git:        gitInfo,
	}

	if uploadDryRun {
		fmt.Printf("• %s\n", path)
		fmt.Printf("  Title: %s\n", title)
		fmt.Printf("  Context: %s | Visibility: %s\n", project, visibility)
		if len(tags) > 0 {
			fmt.Printf("  Tags: %s\n", strings.Join(tags, ", "))
		}
		fmt.Printf("  Content: %d bytes\n", len(body))
		return nil
	}

	resp, err := client.CreateEntry(req)
	if err != nil {
		return err
	}

	fmt.Printf("✓ %s → %s (ID: %s)\n", path, resp.Title, resp.ID)
	return nil
}

// parseFrontmatter splits an optional leading YAML frontmatter block (delimited
// by "---" lines) from the markdown body. If no frontmatter is present, the
// whole input is returned as the body with an empty frontmatter struct.
func parseFrontmatter(content string) (frontmatter, string, error) {
	var fm frontmatter

	// Frontmatter must be the very first thing in the file.
	trimmed := strings.TrimLeft(content, "\uFEFF") // strip BOM if present
	if !strings.HasPrefix(trimmed, "---\n") && !strings.HasPrefix(trimmed, "---\r\n") {
		return fm, content, nil
	}

	// Find the closing delimiter.
	lines := strings.Split(trimmed, "\n")
	closing := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == "---" {
			closing = i
			break
		}
	}
	if closing == -1 {
		// Opening delimiter but no closing one: treat as plain content.
		return fm, content, nil
	}

	yamlBlock := strings.Join(lines[1:closing], "\n")
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return fm, content, err
	}

	body := strings.Join(lines[closing+1:], "\n")
	return fm, body, nil
}

// extractMarkdownTitle returns the text of the first level-1 ATX heading
// (a line beginning with "# "), or "" if none is found.
func extractMarkdownTitle(body string) string {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
		}
	}
	return ""
}

// titleFromFilename derives a readable title from a file path by stripping the
// directory and extension and replacing separators with spaces.
func titleFromFilename(path string) string {
	base := filepath.Base(path)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = strings.NewReplacer("-", " ", "_", " ").Replace(base)
	return strings.TrimSpace(base)
}

// resolve picks the effective value: an explicitly-set flag wins, then the
// per-file frontmatter value, then the fallback (usually a config default).
func resolve(cmd *cobra.Command, flag, flagVal, fmVal, fallback string) string {
	if cmd.Flags().Changed(flag) && flagVal != "" {
		return flagVal
	}
	if fmVal != "" {
		return fmVal
	}
	return fallback
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func init() {
	rootCmd.AddCommand(uploadCmd)

	uploadCmd.Flags().StringVarP(&uploadTitle, "title", "t", "", "Entry title (single file only; overrides frontmatter/heading)")
	uploadCmd.Flags().StringVarP(&uploadProject, "project", "p", "", "Project context: job, project, fun (default from config)")
	uploadCmd.Flags().StringVarP(&uploadMood, "mood", "m", "", "Mood emoji or text")
	uploadCmd.Flags().StringSliceVar(&uploadTags, "tags", nil, "Tags (comma-separated)")
	uploadCmd.Flags().StringVarP(&uploadVisibility, "visibility", "v", "", "Visibility: private or public (default from config)")
	uploadCmd.Flags().BoolVar(&uploadIncludeGit, "include-git", false, "Include git context (remote, branch)")
	uploadCmd.Flags().StringVar(&uploadSource, "source", "", "Source/agent identifier (default: cli)")
	uploadCmd.Flags().StringVar(&uploadRepo, "repo", "", "Repository name (auto-detected from git if not set)")
	uploadCmd.Flags().BoolVar(&uploadDryRun, "dry-run", false, "Preview entries without creating them")
}
