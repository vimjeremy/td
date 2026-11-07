package ui

import (
	"testing"
	"time"

	"github.com/jeremy/td/internal/model"
)

func TestBucket(t *testing.T) {
	now := time.Date(2026, 11, 4, 9, 0, 0, 0, time.Local) // Wednesday
	cases := map[string]int{
		"2026-11-03": gOverdue,
		"2026-11-04": gToday,
		"2026-11-05": gTomorrow,
		"2026-11-08": gThisWeek, // Sunday ends the week
		"2026-11-09": gNextWeek, // next Monday
		"2026-11-15": gNextWeek,
		"2026-11-16": gThisMonth,
		"2026-11-30": gThisMonth,
		"2026-12-01": gNextMonth,
		"2026-12-31": gNextMonth,
		"2027-01-01": gNextYear,
		"2027-12-31": gNextYear,
		"2028-01-01": gLater,
	}
	for s, want := range cases {
		d, _ := time.ParseInLocation("2006-01-02", s, time.Local)
		if got := bucket(model.Task{Due: &d}, now); got != want {
			t.Errorf("%s: group %s, want %s", s, groupNames[got], groupNames[want])
		}
	}
	if bucket(model.Task{}, now) != gNoDate {
		t.Error("undated task should be No date")
	}
	done := time.Now()
	if bucket(model.Task{CompletedAt: &done}, now) != gDone {
		t.Error("finished task should be Done")
	}
	// December: the rest of the year is "This year" only after next month ends, so
	// from late November it runs Dec 1..Dec 31 as next month, and 2027 as next year.
	late := time.Date(2026, 12, 28, 0, 0, 0, 0, time.Local) // Monday
	jan, _ := time.ParseInLocation("2006-01-02", "2027-01-03", time.Local)
	if g := bucket(model.Task{Due: &jan}, late); g != gThisWeek {
		t.Errorf("Jan 3 from Dec 28: %s, want This week", groupNames[g])
	}
}
