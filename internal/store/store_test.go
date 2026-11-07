package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/jeremy/td/internal/model"
)

func open(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "td.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func day(s string) *time.Time {
	d, _ := time.ParseInLocation("2006-01-02", s, time.Local)
	return &d
}

func TestCRUDAndFilters(t *testing.T) {
	s := open(t)
	a := &model.Task{Title: "a", Project: "home", Tags: []string{"x", "y"}, Due: day("2026-11-10")}
	b := &model.Task{Title: "b", Due: day("2026-11-05")}
	c := &model.Task{Title: "c"}
	for _, task := range []*model.Task{a, b, c} {
		if err := s.Add(task); err != nil {
			t.Fatal(err)
		}
	}
	all, _ := s.List(model.Filter{})
	if got := []string{all[0].Title, all[1].Title, all[2].Title}; got[0] != "b" || got[1] != "a" || got[2] != "c" {
		t.Errorf("order = %v; want b a c (due asc, undated last)", got)
	}
	if l, _ := s.List(model.Filter{Project: "home"}); len(l) != 1 || l[0].Title != "a" || len(l[0].Tags) != 2 {
		t.Errorf("project filter: %+v", l)
	}
	if l, _ := s.List(model.Filter{Tag: "y"}); len(l) != 1 {
		t.Errorf("tag filter: %+v", l)
	}
	if l, _ := s.List(model.Filter{NoProject: true}); len(l) != 2 {
		t.Errorf("inbox filter: %+v", l)
	}
	if l, _ := s.List(model.Filter{DueTo: day("2026-11-06")}); len(l) != 1 || l[0].Title != "b" {
		t.Errorf("overdue filter: %+v", l)
	}

	if err := s.SetDone(b.ID, true); err != nil {
		t.Fatal(err)
	}
	if l, _ := s.List(model.Filter{}); len(l) != 2 {
		t.Errorf("done task should be hidden: %+v", l)
	}
	if l, _ := s.List(model.Filter{DoneOnly: true}); len(l) != 1 || !l[0].Done() {
		t.Errorf("done filter: %+v", l)
	}
	s.SetDone(b.ID, false)

	a.Title, a.Tags, a.Due = "a2", []string{"z"}, nil
	if err := s.Update(a); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get(a.ID)
	if got.Title != "a2" || len(got.Tags) != 1 || got.Tags[0] != "z" || got.Due != nil {
		t.Errorf("update: %+v", got)
	}

	if ps, _ := s.Projects(); len(ps) != 1 || ps[0].Open != 1 {
		t.Errorf("projects: %+v", ps)
	}
	if err := s.Delete(c.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(c.ID); err != ErrNotFound {
		t.Errorf("second delete = %v; want ErrNotFound", err)
	}
	if err := s.SetDone(999, true); err != ErrNotFound {
		t.Errorf("SetDone missing = %v", err)
	}
}

func TestCountsKeepDoneUntilDeleted(t *testing.T) {
	s := open(t)
	a := &model.Task{Title: "a", Project: "p", Tags: []string{"g"}}
	s.Add(a)
	check := func(open, total int, when string) {
		t.Helper()
		for _, list := range []func() ([]model.Count, error){s.Projects, s.Tags} {
			cs, _ := list()
			if len(cs) != 1 || cs[0].Open != open || cs[0].Total != total {
				t.Errorf("%s: %+v; want open=%d total=%d", when, cs, open, total)
			}
		}
	}
	check(1, 1, "open")
	s.SetDone(a.ID, true)
	check(0, 1, "done")
	s.Delete(a.ID)
	check(0, 0, "deleted")
}

func TestDirRoundTrip(t *testing.T) {
	s := open(t)
	a := &model.Task{Title: "a", Dir: "/srv/app"}
	if err := s.Add(a); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Get(a.ID); got.Dir != "/srv/app" {
		t.Errorf("dir = %q", got.Dir)
	}
	a.Dir = ""
	s.Update(a)
	if got, _ := s.Get(a.ID); got.Dir != "" {
		t.Errorf("dir not cleared: %q", got.Dir)
	}
}
