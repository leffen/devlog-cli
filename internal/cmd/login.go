package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/leffen/devlog-cli/internal/api"
	"github.com/leffen/devlog-cli/internal/config"
	"github.com/leffen/devlog-cli/internal/machine"
	"github.com/spf13/cobra"
)

var (
	loginProject    string
	loginVisibility string
	loginTags       []string
	loginNoPublicIP bool
	loginDryRun     bool
	loginEvent      string
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Record a machine login event",
	Long: `Record a machine login event as a DevLog entry.

The event captures the machine hostname, username, local IP address, public IP
address (when reachable), and OS/arch. This is handy in a multi-machine work
environment to see where and when you started working.

It is designed to be wired into your OS login flow:

  # macOS / Linux — add to ~/.zprofile, ~/.bash_profile, or a login script:
  devlog login >/dev/null 2>&1 &

  # Linux (systemd user service, runs on graphical/SSH login):
  #   ExecStart=/usr/local/bin/devlog login

  # Windows — add to a startup task or PowerShell profile:
  #   devlog login

Public IP lookup uses https://api.ipify.org and degrades gracefully (the field
is simply omitted) when the machine is offline.

Examples:
  # Record a login event
  devlog login

  # Preview without sending (no API key required)
  devlog login --dry-run

  # Skip the public IP lookup (faster, fully offline)
  devlog login --no-public-ip

  # Record a logout instead
  devlog login --event logout`,
	Run: runLogin,
}

func runLogin(cmd *cobra.Command, args []string) {
	info := machine.Gather(!loginNoPublicIP, 5*time.Second)

	event := strings.TrimSpace(loginEvent)
	if event == "" {
		event = "login"
	}

	title := fmt.Sprintf("%s on %s", titleCase(event), hostnameOrUnknown(info.Hostname))
	content := buildLoginContent(event, info)

	if loginDryRun {
		fmt.Println("Dry run - entry not created:")
		fmt.Printf("Title: %s\n\n%s\n", title, content)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
		os.Exit(1)
	}

	if cfg.APIKey == "" {
		fmt.Fprintln(os.Stderr, "Error: API key not configured. Run 'devlog config init' first (or use --dry-run).")
		os.Exit(1)
	}

	project := loginProject
	if project == "" {
		project = cfg.Defaults.Project
	}

	visibility := loginVisibility
	if visibility == "" {
		visibility = cfg.Defaults.Visibility
	}

	tags := loginTags
	if len(tags) == 0 {
		tags = []string{event, "machine"}
	}

	req := &api.CreateEntryRequest{
		Title:      title,
		Content:    content,
		Context:    project,
		Tags:       tags,
		Visibility: visibility,
		Source:     "machine-" + event,
		Type:       "machine-event",
		Machine: &api.MachineInfo{
			EventKind: event,
			Hostname:  info.Hostname,
			Username:  info.Username,
			LocalIP:   info.LocalIP,
			PublicIP:  info.PublicIP,
			OSArch:    fmt.Sprintf("%s/%s", info.OS, info.Arch),
		},
	}

	client, err := api.NewClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	resp, err := client.CreateEntry(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating entry: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ %s event recorded: %s\n", titleCase(event), resp.Title)
	fmt.Printf("  Machine: %s (%s)\n", hostnameOrUnknown(info.Hostname), info.Username)
	if info.LocalIP != "" {
		fmt.Printf("  Local IP:  %s\n", info.LocalIP)
	}
	if info.PublicIP != "" {
		fmt.Printf("  Public IP: %s\n", info.PublicIP)
	}
}

func buildLoginContent(event string, info machine.Info) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**%s event** on `%s`\n\n", titleCase(event), hostnameOrUnknown(info.Hostname))
	fmt.Fprintf(&b, "| Field | Value |\n")
	fmt.Fprintf(&b, "| --- | --- |\n")
	fmt.Fprintf(&b, "| Time | %s |\n", info.Time.Format(time.RFC3339))
	fmt.Fprintf(&b, "| Hostname | %s |\n", hostnameOrUnknown(info.Hostname))
	fmt.Fprintf(&b, "| Username | %s |\n", orDash(info.Username))
	fmt.Fprintf(&b, "| Local IP | %s |\n", orDash(info.LocalIP))
	fmt.Fprintf(&b, "| Public IP | %s |\n", orDash(info.PublicIP))
	fmt.Fprintf(&b, "| OS / Arch | %s/%s |\n", info.OS, info.Arch)
	return b.String()
}

// titleCase upper-cases the first rune of s, leaving the rest unchanged.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func hostnameOrUnknown(h string) string {
	if h == "" {
		return "unknown-host"
	}
	return h
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func init() {
	rootCmd.AddCommand(loginCmd)

	loginCmd.Flags().StringVarP(&loginProject, "project", "p", "", "Project context: job, project, fun (default from config)")
	loginCmd.Flags().StringVarP(&loginVisibility, "visibility", "v", "", "Visibility: private or public (default from config)")
	loginCmd.Flags().StringSliceVar(&loginTags, "tags", nil, "Tags (comma-separated, default: <event>,machine)")
	loginCmd.Flags().BoolVar(&loginNoPublicIP, "no-public-ip", false, "Skip the public IP lookup")
	loginCmd.Flags().BoolVar(&loginDryRun, "dry-run", false, "Preview the entry without creating it")
	loginCmd.Flags().StringVar(&loginEvent, "event", "login", "Event type label (e.g. login, logout)")
}
