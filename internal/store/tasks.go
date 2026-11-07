package store

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jeremy/td/internal/model"
)

const (
	dayLayout = "2006-01-02"
	tsLayout  = "2006-01-02 15:04:05"
)

// ErrNotFound is returned when a task id does not exist.
var ErrNotFound = errors.New("task not found")

const selectTasks = `
SELECT t.id, t.title, t.notes, COALESCE(p.name, ''),
       CAST(t.due AS TEXT), CAST(t.created_at AS TEXT), CAST(t.completed_at AS TEXT),
       COALESCE((SELECT group_concat(g.name, ',')
                 FROM task_tags x JOIN tags g ON g.id = x.tag_id
                 WHERE x.task_id = t.id), '')
FROM tasks t LEFT JOIN projects p ON p.id = t.project_id`

type execer interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
}

func dueArg(d *time.Time) any {
	if d == nil {
		return nil
	}
	return d.Format(dayLayout)
}

func projectID(tx execer, name string) (any, error) {
	if name == "" {
		return nil, nil
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO projects(name) VALUES(?)`, name); err != nil {
		return nil, err
	}
	var id int64
	err := tx.QueryRow(`SELECT id FROM projects WHERE name = ?`, name).Scan(&id)
	return id, err
}

func setTags(tx execer, taskID int64, tags []string) error {
	if _, err := tx.Exec(`DELETE FROM task_tags WHERE task_id = ?`, taskID); err != nil {
		return err
	}
	for _, name := range tags {
		if _, err := tx.Exec(`INSERT OR IGNORE INTO tags(name) VALUES(?)`, name); err != nil {
			return err
		}
		var tagID int64
		if err := tx.QueryRow(`SELECT id FROM tags WHERE name = ?`, name).Scan(&tagID); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO task_tags(task_id, tag_id) VALUES(?, ?)`, taskID, tagID); err != nil {
			return err
		}
	}
	return nil
}

// Add inserts t and fills in t.ID.
func (s *Store) Add(t *model.Task) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	pid, err := projectID(tx, t.Project)
	if err != nil {
		return err
	}
	res, err := tx.Exec(
		`INSERT INTO tasks(title, notes, project_id, due) VALUES(?, ?, ?, ?)`,
		t.Title, t.Notes, pid, dueArg(t.Due))
	if err != nil {
		return err
	}
	if t.ID, err = res.LastInsertId(); err != nil {
		return err
	}
	if err := setTags(tx, t.ID, t.Tags); err != nil {
		return err
	}
	return tx.Commit()
}

// Update overwrites the editable fields of the task with t.ID.
func (s *Store) Update(t *model.Task) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	pid, err := projectID(tx, t.Project)
	if err != nil {
		return err
	}
	res, err := tx.Exec(
		`UPDATE tasks SET title = ?, notes = ?, project_id = ?, due = ? WHERE id = ?`,
		t.Title, t.Notes, pid, dueArg(t.Due), t.ID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	if err := setTags(tx, t.ID, t.Tags); err != nil {
		return err
	}
	return tx.Commit()
}

// SetDone marks a task complete (or reopens it).
func (s *Store) SetDone(id int64, done bool) error {
	q := `UPDATE tasks SET completed_at = NULL WHERE id = ?`
	if done {
		q = `UPDATE tasks SET completed_at = CURRENT_TIMESTAMP WHERE id = ?`
	}
	res, err := s.db.Exec(q, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Delete removes a task.
func (s *Store) Delete(id int64) error {
	res, err := s.db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Get returns one task by id.
func (s *Store) Get(id int64) (model.Task, error) {
	t, err := scanTask(s.db.QueryRow(selectTasks+` WHERE t.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return t, ErrNotFound
	}
	return t, err
}

// List returns tasks matching f, ordered by due date (undated last), then
// id. Completed tasks sort after open ones.
func (s *Store) List(f model.Filter) ([]model.Task, error) {
	var where []string
	var args []any
	switch {
	case f.DoneOnly:
		where = append(where, "t.completed_at IS NOT NULL")
	case !f.All:
		where = append(where, "t.completed_at IS NULL")
	}
	if f.Project != "" {
		where = append(where, "p.name = ?")
		args = append(args, f.Project)
	}
	if f.NoProject {
		where = append(where, "t.project_id IS NULL")
	}
	if f.Tag != "" {
		where = append(where, `EXISTS (SELECT 1 FROM task_tags x JOIN tags g ON g.id = x.tag_id
			WHERE x.task_id = t.id AND g.name = ?)`)
		args = append(args, f.Tag)
	}
	if f.HasDue {
		where = append(where, "t.due IS NOT NULL")
	}
	if f.DueFrom != nil {
		where = append(where, "t.due >= ?")
		args = append(args, f.DueFrom.Format(dayLayout))
	}
	if f.DueTo != nil {
		where = append(where, "t.due <= ?")
		args = append(args, f.DueTo.Format(dayLayout))
	}
	q := selectTasks
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	if f.DoneOnly {
		q += " ORDER BY t.completed_at DESC, t.id DESC" // most recently finished first
	} else {
		q += " ORDER BY t.completed_at IS NOT NULL, t.due IS NULL, t.due, t.id"
	}

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Projects lists projects with their task counts.
func (s *Store) Projects() ([]model.Count, error) {
	return s.counts(`SELECT p.name, COALESCE(SUM(t.id IS NOT NULL AND t.completed_at IS NULL), 0), COUNT(t.id)
		FROM projects p LEFT JOIN tasks t ON t.project_id = p.id
		GROUP BY p.id ORDER BY p.name`)
}

// Tags lists tags with their task counts.
func (s *Store) Tags() ([]model.Count, error) {
	return s.counts(`SELECT g.name, COALESCE(SUM(t.id IS NOT NULL AND t.completed_at IS NULL), 0), COUNT(t.id)
		FROM tags g
		LEFT JOIN task_tags x ON x.tag_id = g.id
		LEFT JOIN tasks t ON t.id = x.task_id
		GROUP BY g.id ORDER BY g.name`)
}

func (s *Store) counts(q string) ([]model.Count, error) {
	rows, err := s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Count
	for rows.Next() {
		var c model.Count
		if err := rows.Scan(&c.Name, &c.Open, &c.Total); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

type scanner interface{ Scan(dest ...any) error }

func scanTask(r scanner) (model.Task, error) {
	var (
		t                  model.Task
		due, created, done sql.NullString
		tags               string
	)
	if err := r.Scan(&t.ID, &t.Title, &t.Notes, &t.Project,
		&due, &created, &done, &tags); err != nil {
		return t, err
	}
	if due.Valid {
		d, err := time.ParseInLocation(dayLayout, due.String, time.Local)
		if err != nil {
			return t, fmt.Errorf("task %d: bad due date %q", t.ID, due.String)
		}
		t.Due = &d
	}
	if created.Valid {
		t.CreatedAt, _ = time.ParseInLocation(tsLayout, created.String, time.UTC)
	}
	if done.Valid {
		c, _ := time.ParseInLocation(tsLayout, done.String, time.UTC)
		t.CompletedAt = &c
	}
	if tags != "" {
		t.Tags = strings.Split(tags, ",")
		sort.Strings(t.Tags)
	}
	return t, nil
}
