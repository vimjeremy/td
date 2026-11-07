// Package model holds the plain data types shared by the store, parser and UIs.
package model

import "time"

// Task is a single todo item. Due and CompletedAt are nil when unset.
type Task struct {
	ID          int64
	Title       string
	Notes       string
	Dir         string // directory the task relates to; empty when unset
	Project     string // empty means "inbox"
	Due         *time.Time
	Tags        []string
	CreatedAt   time.Time
	CompletedAt *time.Time
}

// Done reports whether the task has been completed.
func (t Task) Done() bool { return t.CompletedAt != nil }

// Filter selects tasks for listing. The zero value means "all open tasks".
type Filter struct {
	Project   string
	Tag       string
	NoProject bool       // only tasks without a project
	HasDue    bool       // only tasks that have a due date
	DueFrom   *time.Time // inclusive lower bound on due date
	DueTo     *time.Time // inclusive upper bound on due date
	All       bool       // include completed tasks
	DoneOnly  bool       // only completed tasks
}

// Count is a project or tag name with its task counts.
type Count struct {
	Name  string
	Open  int // tasks not yet done
	Total int // open and done; 0 means nothing uses it any more
}

// Day truncates t to midnight in its own location.
func Day(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
