// Package parse turns quick-add text into structured tasks.
package parse

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jeremy/td/internal/model"
)

var weekdays = map[string]time.Weekday{
	"sun": time.Sunday, "sunday": time.Sunday,
	"mon": time.Monday, "monday": time.Monday,
	"tue": time.Tuesday, "tues": time.Tuesday, "tuesday": time.Tuesday,
	"wed": time.Wednesday, "wednesday": time.Wednesday,
	"thu": time.Thursday, "thur": time.Thursday, "thurs": time.Thursday, "thursday": time.Thursday,
	"fri": time.Friday, "friday": time.Friday,
	"sat": time.Saturday, "saturday": time.Saturday,
}

var relative = regexp.MustCompile(`^\+(\d+)([dw])$`)

// Date parses today, tomorrow, a weekday name (the next occurrence strictly
// after today), +Nd, +Nw, or YYYY-MM-DD, relative to now.
func Date(s string, now time.Time) (time.Time, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	today := model.Day(now)
	switch s {
	case "today", "tod":
		return today, nil
	case "tomorrow", "tom", "tmrw":
		return today.AddDate(0, 0, 1), nil
	}
	if m := relative.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		if m[2] == "w" {
			n *= 7
		}
		return today.AddDate(0, 0, n), nil
	}
	if wd, ok := weekdays[s]; ok {
		diff := (int(wd) - int(today.Weekday()) + 7) % 7
		if diff == 0 {
			diff = 7
		}
		return today.AddDate(0, 0, diff), nil
	}
	if d, err := time.ParseInLocation("2006-01-02", s, now.Location()); err == nil {
		return d, nil
	}
	return time.Time{}, fmt.Errorf("cannot understand date %q (try today, tomorrow, fri, +3d, +2w, 2026-12-01)", s)
}
