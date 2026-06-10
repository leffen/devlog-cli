package punch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSanitizeProject(t *testing.T) {
	cases := map[string]string{
		"devlog-cli":        "devlog-cli",
		"My Project":        "my-project",
		"  Trailing  ":      "trailing",
		"Foo_Bar.Baz":       "foo-bar-baz",
		"weird!!!chars???":  "weird-chars",
		"---leading-dash--": "leading-dash",
		"":                  "",
	}
	for in, want := range cases {
		if got := SanitizeProject(in); got != want {
			t.Errorf("SanitizeProject(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestProjectFromDir(t *testing.T) {
	if got := ProjectFromDir("/Users/me/p/My Cool App"); got != "my-cool-app" {
		t.Errorf("ProjectFromDir = %q, want my-cool-app", got)
	}
}

func TestAppendAndLoad(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	recs := []*Record{
		{Timestamp: time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), Machine: "m1", Project: "alpha"},
		{Timestamp: time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC), Machine: "m2", Project: "beta", Comment: "note"},
	}
	for _, r := range recs {
		if err := Append(r); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	// Store should live under <config>/devlog/punches.jsonl.
	if _, err := os.Stat(filepath.Join(tmp, "devlog", FileName)); err != nil {
		t.Fatalf("store not created: %v", err)
	}

	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Load returned %d records, want 2", len(got))
	}
	if got[0].Project != "alpha" || got[1].Project != "beta" {
		t.Errorf("records out of order: %+v", got)
	}
	if got[1].Comment != "note" {
		t.Errorf("comment not preserved: %q", got[1].Comment)
	}
}

func TestLoadMissingStoreIsEmpty(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	got, err := Load()
	if err != nil {
		t.Fatalf("Load on missing store: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %d", len(got))
	}
}

func TestNewRecordDerivesProjectFromDir(t *testing.T) {
	rec, err := NewRecord("", "hello")
	if err != nil {
		t.Fatalf("NewRecord: %v", err)
	}
	if rec.Project == "" {
		t.Error("expected derived project name, got empty")
	}
	if rec.Comment != "hello" {
		t.Errorf("comment = %q, want hello", rec.Comment)
	}
	if rec.Dir == "" {
		t.Error("expected dir to be set")
	}
}
