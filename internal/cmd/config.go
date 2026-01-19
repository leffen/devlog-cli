package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/leffen/devlog-cli/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage DevLog CLI configuration",
	Long: `Manage DevLog CLI configuration.

Configuration is stored in ~/.config/devlog/config.yaml by default.
You can also create a .devlog.yaml in your project directory for
project-specific settings.

Available settings:
  api-key     Your DevLog API key (required)
  server      DevLog server URL (default: https://devlog.asd09.com)
  project     Default project context: job, project, or fun (default: job)
  visibility  Default entry visibility: private or public (default: private)
  include-git Default for including git context: true or false (default: true)`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration interactively",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("DevLog CLI Configuration")
		fmt.Println("========================")
		fmt.Println()

		reader := bufio.NewReader(os.Stdin)

		// API Key
		fmt.Print("Enter your API key: ")
		apiKey, _ := reader.ReadString('\n')
		apiKey = strings.TrimSpace(apiKey)
		if apiKey == "" {
			fmt.Println("Error: API key is required")
			os.Exit(1)
		}

		// Server (optional)
		fmt.Printf("Server URL [%s]: ", config.DefaultServer)
		server, _ := reader.ReadString('\n')
		server = strings.TrimSpace(server)
		if server == "" {
			server = config.DefaultServer
		}

		// Default project
		fmt.Print("Default project (job/project/fun) [job]: ")
		project, _ := reader.ReadString('\n')
		project = strings.TrimSpace(project)
		if project == "" {
			project = "job"
		}
		if project != "job" && project != "project" && project != "fun" {
			fmt.Println("Error: Invalid project. Must be job, project, or fun")
			os.Exit(1)
		}

		// Save config
		cfg := &config.Config{
			APIKey: apiKey,
			Server: server,
			Defaults: config.Defaults{
				Project:    project,
				Visibility: "private",
				IncludeGit: true,
			},
		}

		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %s\n", err)
			os.Exit(1)
		}

		configPath, _ := config.GetConfigPath()
		fmt.Println()
		fmt.Printf("Configuration saved to %s\n", configPath)
		fmt.Println("You're ready to use DevLog CLI!")
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Available keys:
  api-key     Your DevLog API key
  server      DevLog server URL
  project     Default project context (job, project, fun)
  visibility  Default entry visibility (private, public)
  include-git Include git context by default (true, false)`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		value := args[1]

		if err := config.Set(key, value); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("Set %s = %s\n", key, value)
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]

		value, err := config.Get(key)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}

		// Mask API key for security
		if key == "api-key" || key == "apikey" || key == "api_key" {
			if len(value) > 12 {
				value = value[:12] + "..."
			}
		}

		fmt.Println(value)
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %s\n", err)
			os.Exit(1)
		}

		// Mask API key
		apiKeyDisplay := "(not set)"
		if cfg.APIKey != "" {
			if len(cfg.APIKey) > 12 {
				apiKeyDisplay = cfg.APIKey[:12] + "..."
			} else {
				apiKeyDisplay = cfg.APIKey + "..."
			}
		}

		fmt.Println("DevLog CLI Configuration")
		fmt.Println("========================")
		fmt.Printf("API Key:    %s\n", apiKeyDisplay)
		fmt.Printf("Server:     %s\n", cfg.Server)
		fmt.Println()
		fmt.Println("Defaults:")
		fmt.Printf("  Project:    %s\n", cfg.Defaults.Project)
		fmt.Printf("  Visibility: %s\n", cfg.Defaults.Visibility)
		fmt.Printf("  Include Git: %v\n", cfg.Defaults.IncludeGit)
		fmt.Println()

		configPath, _ := config.GetConfigPath()
		fmt.Printf("Config file: %s\n", configPath)
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	Run: func(cmd *cobra.Command, args []string) {
		configPath, err := config.GetConfigPath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		fmt.Println(configPath)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
}
