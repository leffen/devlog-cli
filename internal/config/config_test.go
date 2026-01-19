package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	if DefaultServer != "https://devlog.asd09.com" {
		t.Errorf("expected default server to be https://devlog.asd09.com, got %s", DefaultServer)
	}
	if DefaultProject != "job" {
		t.Errorf("expected default project to be job, got %s", DefaultProject)
	}
}

func TestGetConfigDir(t *testing.T) {
	// Test with XDG_CONFIG_HOME set
	t.Run("with XDG_CONFIG_HOME", func(t *testing.T) {
		oldXDG := os.Getenv("XDG_CONFIG_HOME")
		defer os.Setenv("XDG_CONFIG_HOME", oldXDG)

		os.Setenv("XDG_CONFIG_HOME", "/tmp/test-config")
		dir, err := GetConfigDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "/tmp/test-config/devlog"
		if dir != expected {
			t.Errorf("expected %s, got %s", expected, dir)
		}
	})

	// Test without XDG_CONFIG_HOME (falls back to ~/.config)
	t.Run("without XDG_CONFIG_HOME", func(t *testing.T) {
		oldXDG := os.Getenv("XDG_CONFIG_HOME")
		defer os.Setenv("XDG_CONFIG_HOME", oldXDG)

		os.Unsetenv("XDG_CONFIG_HOME")
		dir, err := GetConfigDir()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		home, _ := os.UserHomeDir()
		expected := filepath.Join(home, ".config", "devlog")
		if dir != expected {
			t.Errorf("expected %s, got %s", expected, dir)
		}
	})
}

func TestGetProjectConfigPath(t *testing.T) {
	path := GetProjectConfigPath()
	if path != ".devlog.yaml" {
		t.Errorf("expected .devlog.yaml, got %s", path)
	}
}

func TestLoadWithDefaults(t *testing.T) {
	// Create a temp directory for config
	tmpDir := t.TempDir()

	// Set XDG_CONFIG_HOME to temp dir
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Change to temp dir to avoid loading project config
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check defaults
	if cfg.Server != DefaultServer {
		t.Errorf("expected server %s, got %s", DefaultServer, cfg.Server)
	}
	if cfg.Defaults.Project != DefaultProject {
		t.Errorf("expected project %s, got %s", DefaultProject, cfg.Defaults.Project)
	}
	if cfg.Defaults.Visibility != "private" {
		t.Errorf("expected visibility private, got %s", cfg.Defaults.Visibility)
	}
	if !cfg.Defaults.IncludeGit {
		t.Error("expected include_git to be true by default")
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()

	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create and save config
	cfg := &Config{
		APIKey: "test-api-key",
		Server: "https://custom.server.com",
		Defaults: Defaults{
			Project:    "fun",
			Visibility: "public",
			IncludeGit: false,
		},
	}

	err := Save(cfg)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Load and verify
	loaded, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.APIKey != cfg.APIKey {
		t.Errorf("expected api_key %s, got %s", cfg.APIKey, loaded.APIKey)
	}
	if loaded.Server != cfg.Server {
		t.Errorf("expected server %s, got %s", cfg.Server, loaded.Server)
	}
	if loaded.Defaults.Project != cfg.Defaults.Project {
		t.Errorf("expected project %s, got %s", cfg.Defaults.Project, loaded.Defaults.Project)
	}
	if loaded.Defaults.Visibility != cfg.Defaults.Visibility {
		t.Errorf("expected visibility %s, got %s", cfg.Defaults.Visibility, loaded.Defaults.Visibility)
	}
	if loaded.Defaults.IncludeGit != cfg.Defaults.IncludeGit {
		t.Errorf("expected include_git %v, got %v", cfg.Defaults.IncludeGit, loaded.Defaults.IncludeGit)
	}
}

func TestSetAndGet(t *testing.T) {
	tmpDir := t.TempDir()

	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	tests := []struct {
		key      string
		value    string
		expected string
	}{
		{"api-key", "my-test-key", "my-test-key"},
		{"server", "https://test.com", "https://test.com"},
		{"project", "fun", "fun"},
		{"visibility", "public", "public"},
		{"include-git", "false", "false"},
	}

	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			err := Set(tc.key, tc.value)
			if err != nil {
				t.Fatalf("failed to set %s: %v", tc.key, err)
			}

			got, err := Get(tc.key)
			if err != nil {
				t.Fatalf("failed to get %s: %v", tc.key, err)
			}

			if got != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, got)
			}
		})
	}
}

func TestSetInvalidValues(t *testing.T) {
	tmpDir := t.TempDir()

	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	tests := []struct {
		key   string
		value string
	}{
		{"project", "invalid"},
		{"visibility", "invalid"},
		{"unknown-key", "value"},
	}

	for _, tc := range tests {
		t.Run(tc.key+"="+tc.value, func(t *testing.T) {
			err := Set(tc.key, tc.value)
			if err == nil {
				t.Errorf("expected error for %s=%s, got nil", tc.key, tc.value)
			}
		})
	}
}

func TestGetUnknownKey(t *testing.T) {
	tmpDir := t.TempDir()

	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	_, err := Get("unknown-key")
	if err == nil {
		t.Error("expected error for unknown key, got nil")
	}
}

func TestIsConfigured(t *testing.T) {
	tmpDir := t.TempDir()

	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)
	os.Setenv("XDG_CONFIG_HOME", tmpDir)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Should not be configured initially
	if IsConfigured() {
		t.Error("expected IsConfigured to return false initially")
	}

	// Set API key
	Set("api-key", "test-key")

	// Should be configured now
	if !IsConfigured() {
		t.Error("expected IsConfigured to return true after setting api-key")
	}
}
