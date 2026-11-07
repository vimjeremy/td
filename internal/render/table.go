// Package render formats tasks for the terminal. Colors are applied through
// lipgloss, which disables them automatically when stdout is not a TTY or
// NO_COLOR is set.
package render

import (
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jeremy/td/internal/model"
)

var (
	Red    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	Yellow = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	Dim    = lipgloss.NewStyle().Faint(true)
	Bold   = lipgloss.NewStyle().Bold(true)
	Strike = lipgloss.NewStyle().Faint(true).Strikethrough(true)
)

// dueDays is the whole number of calendar days from now to due (negative when overdue).
func dueDays(due *time.Time, now time.Time) int {
	return int(math.Round(model.Day(*due).Sub(model.Day(now)).Hours() / 24)) // Round absorbs DST 23h/25h days
}

// RelativeLabel describes a due date only relative to today: "today",
// "tomorrow", "in 5 days", "yesterday", "3 days ago".
func RelativeLabel(due *time.Time, now time.Time) string {
	if due == nil {
		return ""
	}
	switch d := dueDays(due, now); {
	case d == 0:
		return "today"
	case d == 1:
		return "tomorrow"
	case d == -1:
		return "yesterday"
	case d < 0:
		return fmt.Sprintf("%d days ago", -d)
	default:
		return fmt.Sprintf("in %d days", d)
	}
}

// DueLabel is a short human label for a due date.
func DueLabel(due *time.Time, now time.Time) string {
	if due == nil {
		return ""
	}
	days := dueDays(due, now)
	date := due.Format("Jan 2")
	if due.Year() != now.Year() {
		date = due.Format("Jan 2 2006")
	}
	switch {
	case days < 0:
		return fmt.Sprintf("-%dd %s", -days, date)
	case days == 0:
		return "today " + date
	case days == 1:
		return "tomorrow " + date
	case days < 7:
		return due.Format("Mon ") + date
	default:
		return date
	}
}

// DueStyle colors a due label: red when overdue, yellow when due today.
func DueStyle(due *time.Time, now time.Time) lipgloss.Style {
	if due == nil {
		return lipgloss.NewStyle()
	}
	switch d, n := model.Day(*due), model.Day(now); {
	case d.Before(n):
		return Red
	case d.Equal(n):
		return Yellow
	}
	return lipgloss.NewStyle()
}

func pad(s string, w int) string {
	if n := w - lipgloss.Width(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

// Table writes tasks as an aligned table.
func Table(w io.Writer, tasks []model.Task, now time.Time) {
	head := []string{"ID", "DUE", "PROJECT", "TAGS"}
	width := make([]int, len(head))
	for i, h := range head {
		width[i] = len(h)
	}
	cells := make([][]string, len(tasks))
	for i, t := range tasks {
		tags := ""
		if len(t.Tags) > 0 {
			tags = "@" + strings.Join(t.Tags, " @")
		}
		proj := ""
		if t.Project != "" {
			proj = "+" + t.Project
		}
		cells[i] = []string{fmt.Sprint(t.ID), DueLabel(t.Due, now), proj, tags}
		for c, v := range cells[i] {
			width[c] = max(width[c], lipgloss.Width(v))
		}
	}
	var b strings.Builder
	for i, h := range head {
		b.WriteString(Dim.Underline(true).Render(pad(h, width[i])) + "  ")
	}
	b.WriteString(Dim.Underline(true).Render("TITLE") + "\n")
	for i, t := range tasks {
		c := cells[i]
		b.WriteString(pad(c[0], width[0]) + "  ")
		b.WriteString(DueStyle(t.Due, now).Render(pad(c[1], width[1])) + "  ")
		b.WriteString(pad(c[2], width[2]) + "  ")
		b.WriteString(Dim.Render(pad(c[3], width[3])) + "  ")
		if t.Done() {
			b.WriteString(Strike.Render(t.Title))
		} else {
			b.WriteString(t.Title)
		}
		b.WriteString("\n")
	}
	io.WriteString(w, b.String())
}
