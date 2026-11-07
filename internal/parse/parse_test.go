package parse

import (
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
