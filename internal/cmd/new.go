package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/leffen/devlog-cli/internal/api"
	"github.com/leffen/devlog-cli/internal/config"
	"github.com/leffen/devlog-cli/internal/git"
	"github.com/spf13/cobra"
)

var (
	newTitle      string
	newContent    string
	newFile       string
	newProject    string
	newMood       string
	newTags       []string
	newVisibility string
	newIncludeGit bool
	newEditor     bool
	newAutoTitle  bool
	newStdin      bool
)

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new DevLog entry",
	Long: `Create a new DevLog entry.

Content can be provided via:
  - The -c/--content flag
  - A file with -f/--file
  - Standard input with - (use "devlog new -t 'Title' -")
  - An editor with --editor

Examples:
  # Simple entry with flags
  devlog new -t "Fixed auth bug" -c "Updated login flow to handle edge cases"

  # From stdin (great for Claude Code)
  echo "Implemented OAuth2 login flow" | devlog new -t "Auth Implementation" -

  # From a file
  devlog new -t "Session Notes" -f notes.md

  # With git context
  devlog new -t "Feature complete" -c "Finished the search feature" --include-git

  # Open editor
  devlog new --editor

  # With tags
  devlog new -t "Bug fix" -c "Fixed memory leak" --tags "bugfix,performance"`,
	Run: runNew,
}

func runNew(cmd *cobra.Command, args []string) {
	// Check if stdin mode
	if len(args) > 0 && args[0] == "-" {
		newStdin = true
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
		os.Exit(1)
	}

	// Check API key
	if cfg.APIKey == "" {
		fmt.Fprintln(os.Stderr, "Error: API key not configured. Run 'devlog config init' first.")
		os.Exit(1)
	}

	// Determine project (use default if not specified)
	if newProject == "" {
		newProject = cfg.Defaults.Project
	}

	// Determine visibility (use default if not specified)
	if newVisibility == "" {
		newVisibility = cfg.Defaults.Visibility
	}

	// Determine include-git (use default if flag not explicitly set)
	if !cmd.Flags().Changed("include-git") {
		newIncludeGit = cfg.Defaults.IncludeGit
	}

	// Get content from various sources
	content := newContent

	if newStdin {
		// Read from stdin
		stdinContent, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %s\n", err)
			os.Exit(1)
		}
		content = strings.TrimSpace(string(stdinContent))
	} else if newFile != "" {
		// Read from file
		fileContent, err := os.ReadFile(newFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %s\n", err)
			os.Exit(1)
		}
		content = strings.TrimSpace(string(fileContent))
	} else if newEditor {
		// Open editor
		content, err = openEditor("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening editor: %s\n", err)
			os.Exit(1)
		}
	}

	// Validate title
	title := newTitle
	if title == "" && newAutoTitle {
		// Auto-generate title from content or git
		title = autoGenerateTitle(content)
	}
	if title == "" {
		fmt.Fprintln(os.Stderr, "Error: Title is required. Use -t/--title or --auto-title")
		os.Exit(1)
	}

	// Validate content
	if content == "" {
		fmt.Fprintln(os.Stderr, "Error: Content is required. Use -c/--content, -f/--file, or stdin")
		os.Exit(1)
	}

	// Build request
	req := &api.CreateEntryRequest{
		Title:      title,
		Content:    content,
		Context:    newProject,
		Mood:       newMood,
		Tags:       newTags,
		Visibility: newVisibility,
	}

	// Add git context if requested
	if newIncludeGit && git.IsInGitRepo() {
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
	fmt.Printf("✓ Entry created: %s\n", resp.Title)
	fmt.Printf("  ID: %s\n", resp.ID)
	fmt.Printf("  Context: %s\n", resp.Context)
	if len(resp.Tags) > 0 {
		fmt.Printf("  Tags: %s\n", strings.Join(resp.Tags, ", "))
	}
}

func autoGenerateTitle(content string) string {
	// Try to extract first line or first sentence
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			// Use first non-empty, non-header line
			if len(line) > 60 {
				return line[:57] + "..."
			}
			return line
		}
	}

	// Fall back to git branch name if available
	if git.IsInGitRepo() {
		if branch, err := git.GetCurrentBranch(); err == nil {
			return "Work on " + branch
		}
	}

	return ""
}

func openEditor(initialContent string) (string, error) {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "devlog-*.md")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write initial content
	if initialContent != "" {
		if _, err := tmpFile.WriteString(initialContent); err != nil {
			return "", fmt.Errorf("failed to write to temp file: %w", err)
		}
	} else {
		// Write template
		template := `# DevLog Entry

<!-- Write your entry content here -->
<!-- Lines starting with <!-- will be removed -->

`
		if _, err := tmpFile.WriteString(template); err != nil {
			return "", fmt.Errorf("failed to write template: %w", err)
		}
	}
	tmpFile.Close()

	// Get editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi" // Fallback
	}

	// Open editor
	editorCmd := exec.Command(editor, tmpFile.Name())
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return "", fmt.Errorf("editor exited with error: %w", err)
	}

	// Read result
	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read temp file: %w", err)
	}

	// Remove comment lines
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(strings.TrimSpace(line), "<!--") {
			lines = append(lines, line)
		}
	}

	return strings.TrimSpace(strings.Join(lines, "\n")), nil
}

func init() {
	rootCmd.AddCommand(newCmd)

	newCmd.Flags().StringVarP(&newTitle, "title", "t", "", "Entry title (required unless --auto-title)")
	newCmd.Flags().StringVarP(&newContent, "content", "c", "", "Entry content")
	newCmd.Flags().StringVarP(&newFile, "file", "f", "", "Read content from file")
	newCmd.Flags().StringVarP(&newProject, "project", "p", "", "Project context: job, project, fun (default from config)")
	newCmd.Flags().StringVarP(&newMood, "mood", "m", "", "Mood emoji or text")
	newCmd.Flags().StringSliceVar(&newTags, "tags", nil, "Tags (comma-separated)")
	newCmd.Flags().StringVarP(&newVisibility, "visibility", "v", "", "Visibility: private or public (default from config)")
	newCmd.Flags().BoolVar(&newIncludeGit, "include-git", false, "Include git context (remote, branch)")
	newCmd.Flags().BoolVar(&newEditor, "editor", false, "Open editor for content")
	newCmd.Flags().BoolVar(&newAutoTitle, "auto-title", false, "Auto-generate title from content")
}
