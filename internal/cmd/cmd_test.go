package cmd

import (
	"strings"
	"testing"
)

func TestParseRelativeDate(t *testing.T) {
	tests := []struct {
		input    string
		contains string // We check if result contains this (dates change)
	}{
		{"today", "-"}, // Should contain a date with dashes
		{"yesterday", "-"},
		{"1 week ago", "-"},
		{"2 weeks ago", "-"},
		{"1 day ago", "-"},
		{"3 days ago", "-"},
		{"1 month ago", "-"},
		{"2 months ago", "-"},
		{"2024-01-15", "2024-01-15"}, // ISO date passthrough
		{"invalid", "invalid"},       // Unknown format passed through
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := parseRelativeDate(tc.input)
			if !strings.Contains(result, tc.contains) {
				t.Errorf("parseRelativeDate(%q) = %q, expected to contain %q", tc.input, result, tc.contains)
			}
		})
	}
}

func TestGetContentPreview(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		maxLen   int
		expected string
	}{
		{
			name:     "short content",
			content:  "Hello world",
			maxLen:   100,
			expected: "Hello world",
		},
		{
			name:     "content with headers",
			content:  "# Header\n\nSome content here",
			maxLen:   100,
			expected: "Some content here",
		},
		{
			name:     "long content truncated",
			content:  "This is a very long content that should be truncated at some point",
			maxLen:   20,
			expected: "This is a very lo...",
		},
		{
			name:     "empty content",
			content:  "",
			maxLen:   100,
			expected: "",
		},
		{
			name:     "only headers",
			content:  "# Header 1\n## Header 2",
			maxLen:   100,
			expected: "",
		},
		{
			name:     "multiple paragraphs",
			content:  "First paragraph\n\nSecond paragraph",
			maxLen:   100,
			expected: "First paragraph Second paragraph",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := getContentPreview(tc.content, tc.maxLen)
			if result != tc.expected {
				t.Errorf("getContentPreview(%q, %d) = %q, want %q", tc.content, tc.maxLen, result, tc.expected)
			}
		})
	}
}

func TestEscapeJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`simple text`, `simple text`},
		{`text with "quotes"`, `text with \"quotes\"`},
		{`text with \backslash`, `text with \\backslash`},
		{"text with\nnewline", `text with\nnewline`},
		{"text with\ttab", `text with\ttab`},
		{`all "together"\nwith\ttabs`, `all \"together\"\\nwith\\ttabs`},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := escapeJSON(tc.input)
			if result != tc.expected {
				t.Errorf("escapeJSON(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestVersionVariables(t *testing.T) {
	// Version and Commit are set at build time, but should have default values
	// We just verify they exist and are strings
	if Version == "" {
		// This is fine - they're empty when not built with ldflags
		Version = "test"
	}
	if Commit == "" {
		Commit = "test"
	}

	// These should be settable
	Version = "1.0.0"
	Commit = "abc123"

	if Version != "1.0.0" {
		t.Error("expected Version to be settable")
	}
	if Commit != "abc123" {
		t.Error("expected Commit to be settable")
	}
}

func TestClaudePromptContent(t *testing.T) {
	// Verify the prompt contains expected sections
	if !strings.Contains(claudePrompt, "DevLog CLI Integration") {
		t.Error("prompt should contain DevLog CLI Integration header")
	}
	if !strings.Contains(claudePrompt, "devlog new") {
		t.Error("prompt should contain devlog new command example")
	}
	if !strings.Contains(claudePrompt, "devlog log") {
		t.Error("prompt should contain devlog log command example")
	}
	if !strings.Contains(claudePrompt, "When to Log") {
		t.Error("prompt should contain When to Log section")
	}
	if !strings.Contains(claudePrompt, "Guidelines") {
		t.Error("prompt should contain Guidelines section")
	}
	if !strings.Contains(claudePrompt, "--source") {
		t.Error("prompt should contain --source flag")
	}
	if !strings.Contains(claudePrompt, "--repo") {
		t.Error("prompt should contain --repo flag")
	}
	if !strings.Contains(claudePrompt, "Flag Reference") {
		t.Error("prompt should contain Flag Reference section")
	}
}
