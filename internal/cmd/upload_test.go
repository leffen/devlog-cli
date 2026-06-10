package cmd

import (
	"reflect"
	"testing"
)

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantTitle string
		wantTags  []string
		wantBody  string
	}{
		{
			name:     "no frontmatter",
			content:  "# Heading\n\nBody",
			wantBody: "# Heading\n\nBody",
		},
		{
			name:      "full frontmatter",
			content:   "---\ntitle: My Entry\ntags: [a, b]\nproject: job\n---\n\n# Heading\n\nBody",
			wantTitle: "My Entry",
			wantTags:  []string{"a", "b"},
			wantBody:  "\n# Heading\n\nBody",
		},
		{
			name:     "opening delimiter without closing",
			content:  "---\nnot really frontmatter\n\nbody",
			wantBody: "---\nnot really frontmatter\n\nbody",
		},
		{
			name:      "crlf line endings",
			content:   "---\r\ntitle: Win\r\n---\r\nbody",
			wantTitle: "Win",
			wantBody:  "body",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fm, body, err := parseFrontmatter(tc.content)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if fm.Title != tc.wantTitle {
				t.Errorf("title = %q, want %q", fm.Title, tc.wantTitle)
			}
			if tc.wantTags != nil && !reflect.DeepEqual(fm.Tags, tc.wantTags) {
				t.Errorf("tags = %v, want %v", fm.Tags, tc.wantTags)
			}
			if body != tc.wantBody {
				t.Errorf("body = %q, want %q", body, tc.wantBody)
			}
		})
	}
}

func TestExtractMarkdownTitle(t *testing.T) {
	tests := []struct {
		body string
		want string
	}{
		{"# Hello World\n\nbody", "Hello World"},
		{"intro line\n\n# Real Title", "Real Title"},
		{"## Subheading only", ""},
		{"no headings here", ""},
		{"#NoSpace is not a heading", ""},
		{"   # Indented Title", "Indented Title"},
	}

	for _, tc := range tests {
		t.Run(tc.body, func(t *testing.T) {
			if got := extractMarkdownTitle(tc.body); got != tc.want {
				t.Errorf("extractMarkdownTitle(%q) = %q, want %q", tc.body, got, tc.want)
			}
		})
	}
}

func TestTitleFromFilename(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"notes.md", "notes"},
		{"daily/my-standup-notes.md", "my standup notes"},
		{"/abs/path/release_notes.markdown", "release notes"},
		{"plain", "plain"},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			if got := titleFromFilename(tc.path); got != tc.want {
				t.Errorf("titleFromFilename(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "", "x", "y"); got != "x" {
		t.Errorf("firstNonEmpty = %q, want %q", got, "x")
	}
	if got := firstNonEmpty("", ""); got != "" {
		t.Errorf("firstNonEmpty = %q, want empty", got)
	}
}
