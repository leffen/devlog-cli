package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DefaultServer  = "https://devlog.asd09.com"
	DefaultProject = "job"
	ConfigFileName = "config.yaml"
	ConfigDirName  = "devlog"
)

// Config represents the CLI configuration
type Config struct {
	APIKey   string   `yaml:"api_key"`
	Server   string   `yaml:"server"`
	Defaults Defaults `yaml:"defaults"`
}

// Defaults represents default values for new entries
type Defaults struct {
	Project    string `yaml:"project"`    // job, project, fun
	Visibility string `yaml:"visibility"` // private, public
	IncludeGit bool   `yaml:"include_git"`
}

// GetConfigDir returns the config directory path
func GetConfigDir() (string, error) {
	// Check XDG_CONFIG_HOME first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, ConfigDirName), nil
	}

	// Fall back to ~/.config
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(home, ".config", ConfigDirName), nil
}

// GetConfigPath returns the full path to the config file
func GetConfigPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ConfigFileName), nil
}

// GetProjectConfigPath returns the path for project-local config
func GetProjectConfigPath() string {
	return ".devlog.yaml"
}

// Load loads the configuration from disk
// It merges global and project-local configs, with local taking precedence
func Load() (*Config, error) {
	cfg := &Config{
		Server: DefaultServer,
		Defaults: Defaults{
			Project:    DefaultProject,
			Visibility: "private",
			IncludeGit: true,
		},
	}

	// Load global config
	globalPath, err := GetConfigPath()
	if err == nil {
		if data, err := os.ReadFile(globalPath); err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("failed to parse global config: %w", err)
			}
		}
	}

	// Load project-local config (overrides global)
	localPath := GetProjectConfigPath()
	if data, err := os.ReadFile(localPath); err == nil {
		localCfg := &Config{}
		if err := yaml.Unmarshal(data, localCfg); err != nil {
			return nil, fmt.Errorf("failed to parse local config: %w", err)
		}
		// Merge local into global (local takes precedence)
		if localCfg.APIKey != "" {
			cfg.APIKey = localCfg.APIKey
		}
		if localCfg.Server != "" {
			cfg.Server = localCfg.Server
		}
		if localCfg.Defaults.Project != "" {
			cfg.Defaults.Project = localCfg.Defaults.Project
		}
		if localCfg.Defaults.Visibility != "" {
			cfg.Defaults.Visibility = localCfg.Defaults.Visibility
		}
		// IncludeGit is a bool, so we just use whatever local says
		cfg.Defaults.IncludeGit = localCfg.Defaults.IncludeGit
	}

	return cfg, nil
}

// Save saves the configuration to disk
func Save(cfg *Config) error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	path := filepath.Join(dir, ConfigFileName)
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Set sets a configuration value
func Set(key, value string) error {
	cfg, err := Load()
	if err != nil {
		cfg = &Config{
			Server: DefaultServer,
			Defaults: Defaults{
				Project:    DefaultProject,
				Visibility: "private",
				IncludeGit: true,
			},
		}
	}

	switch key {
	case "api-key", "apikey", "api_key":
		cfg.APIKey = value
	case "server":
		cfg.Server = value
	case "default-project", "project":
		if value != "job" && value != "project" && value != "fun" {
			return fmt.Errorf("invalid project value: %s (must be job, project, or fun)", value)
		}
		cfg.Defaults.Project = value
	case "default-visibility", "visibility":
		if value != "private" && value != "public" {
			return fmt.Errorf("invalid visibility value: %s (must be private or public)", value)
		}
		cfg.Defaults.Visibility = value
	case "include-git", "include_git":
		cfg.Defaults.IncludeGit = value == "true" || value == "1" || value == "yes"
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return Save(cfg)
}

// Get returns a configuration value
func Get(key string) (string, error) {
	cfg, err := Load()
	if err != nil {
		return "", err
	}

	switch key {
	case "api-key", "apikey", "api_key":
		return cfg.APIKey, nil
	case "server":
		return cfg.Server, nil
	case "default-project", "project":
		return cfg.Defaults.Project, nil
	case "default-visibility", "visibility":
		return cfg.Defaults.Visibility, nil
	case "include-git", "include_git":
		if cfg.Defaults.IncludeGit {
			return "true", nil
		}
		return "false", nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

// IsConfigured returns true if the CLI is configured with an API key
func IsConfigured() bool {
	cfg, err := Load()
	if err != nil {
		return false
	}
	return cfg.APIKey != ""
}
