package ui

import (
	"time"

	"github.com/jeremy/td/internal/model"
)

// Groups the task list is split into, in display order. Open tasks are sorted
// by due date, so each group is contiguous; finished tasks sort last.
const (
	gOverdue = iota
	gToday
	gTomorrow
	gThisWeek
	gNextWeek
	gThisMonth
	gNextMonth
	gThisYear
	gNextYear
	gLater
	gNoDate
	gDone
)

var groupNames = []string{
	"Overdue", "Today", "Tomorrow", "This week", "Next week", "This month",
	"Next month", "This year", "Next year", "Later", "No date", "Done",
}

// bucket returns the group of t relative to now. Weeks run Monday to Sunday.
// A group only holds dates after the end of the previous one, so a week that
// straddles a month end simply leaves "This month" empty.
func bucket(t model.Task, now time.Time) int {
	if t.Done() {
		return gDone
	}
	if t.Due == nil {
		return gNoDate
	}
	today := model.Day(now)
	y, m, d := today.Date()
	loc := today.Location()
	daysSinceMonday := (int(today.Weekday()) + 6) % 7
	endThisWeek := today.AddDate(0, 0, 6-daysSinceMonday)
	due := model.Day(*t.Due)
	cutoffs := []struct {
		group int
		end   time.Time
	}{
		{gThisWeek, endThisWeek},
		{gNextWeek, endThisWeek.AddDate(0, 0, 7)},
		{gThisMonth, time.Date(y, m+1, 0, 0, 0, 0, 0, loc)},
		{gNextMonth, time.Date(y, m+2, 0, 0, 0, 0, 0, loc)},
		{gThisYear, time.Date(y, 12, 31, 0, 0, 0, 0, loc)},
		{gNextYear, time.Date(y+1, 12, 31, 0, 0, 0, 0, loc)},
	}
	switch {
	case due.Before(today):
		return gOverdue
	case due.Equal(today):
		return gToday
	case due.Equal(time.Date(y, m, d+1, 0, 0, 0, 0, loc)):
		return gTomorrow
	}
	for _, c := range cutoffs {
		if !due.After(c.end) {
			return c.group
		}
	}
	return gLater
}
