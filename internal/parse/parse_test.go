package parse

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// 2026-11-07 is a Saturday.
var now = time.Date(2026, 11, 7, 15, 30, 0, 0, time.Local)

func TestDate(t *testing.T) {
	cases := map[string]string{
		"today": "2026-11-07", "tomorrow": "2026-11-08", "sat": "2026-11-14",
		"fri": "2026-11-13", "mon": "2026-11-09", "+3d": "2026-11-10",
		"+2w": "2026-11-21", "2026-12-01": "2026-12-01",
	}
	for in, want := range cases {
		got, err := Date(in, now)
		if err != nil || got.Format("2006-01-02") != want {
			t.Errorf("Date(%q) = %v, %v; want %s", in, got, err, want)
		}
	}
	if _, err := Date("someday", now); err == nil {
		t.Error("expected error for bad date")
	}
}

func TestInput(t *testing.T) {
	p, err := Input([]string{"Call dentist", "+personal", "@phone", "@urgent", "due:tomorrow"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "Call dentist" || p.Project != "personal" || len(p.Tags) != 2 ||
		p.Due.Format("2006-01-02") != "2026-11-08" {
		t.Errorf("unexpected parse: %+v", p)
	}
	// Unknown !word stays in the title; due:none clears.
	p, _ = Input([]string{"Wow !amazing due:none"}, now)
	if p.Title != "Wow !amazing" || !p.ClearDue {
		t.Errorf("unexpected parse: %+v", p)
	}
	if _, err := Input([]string{"x due:nope"}, now); err == nil {
		t.Error("expected error for bad due")
	}
	if _, err := (Parsed{}).ToTask(); err == nil {
		t.Error("expected error for empty title")
	}
}

func TestDirToken(t *testing.T) {
	p, err := Input([]string{"Fix build", "dir:/srv/app"}, now)
	if err != nil || p.Dir != "/srv/app" || !p.HasDir || p.Title != "Fix build" {
		t.Errorf("got %+v, %v", p, err)
	}
	if p, _ = Input([]string{"x dir:none"}, now); !p.HasDir || p.Dir != "" {
		t.Errorf("dir:none should clear: %+v", p)
	}
	home, _ := os.UserHomeDir()
	if d, _ := ExpandDir("~/code"); d != filepath.Join(home, "code") {
		t.Errorf("ExpandDir(~/code) = %q", d)
	}
}

func TestNoteToken(t *testing.T) {
	p, _ := Input([]string{"Call dentist", "+me", "note:", "ask", "about", "due:fri", "+x"}, now)
	if p.Title != "Call dentist" || p.Project != "me" || !p.HasNote || p.Note != "ask about due:fri +x" || p.Due != nil {
		t.Errorf("note must take the rest of the line: %+v", p)
	}
	p, _ = Input([]string{"x note:remember the milk"}, now)
	if p.Title != "x" || p.Note != "remember the milk" {
		t.Errorf("%+v", p)
	}
	if p, _ = Input([]string{"x note:none"}, now); !p.HasNote || p.Note != "" {
		t.Errorf("note:none should clear: %+v", p)
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	t.Setenv("PROJ", "/srv/projects")
	t.Setenv("TD_UNSET_TEST", "")
	os.Unsetenv("TD_UNSET_TEST")
	cases := map[string]string{
		"~":             home,
		"~/code":        filepath.Join(home, "code"),
		"$PROJ/app":     "/srv/projects/app",
		"${PROJ}/app":   "/srv/projects/app",
		"/abs/path":     "/abs/path",
		"relative/dir":  "relative/dir", // only ~ and $VAR are expanded
		"foo/$PROJ":     "foo/$PROJ",    // not at the start: left alone
		"  ~/trimmed  ": filepath.Join(home, "trimmed"),
	}
	for in, want := range cases {
		if got, err := ExpandPath(in); err != nil || got != want {
			t.Errorf("ExpandPath(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := ExpandPath("$TD_UNSET_TEST/x"); err == nil {
		t.Error("unset variable should be an error")
	}
}
