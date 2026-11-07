// Package store persists tasks in a SQLite database.
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

// Store is a handle on the task database.
type Store struct{ db *sql.DB }

// migrations are applied in order; PRAGMA user_version records how many ran.
var migrations = []string{
	`CREATE TABLE projects (
		id   INTEGER PRIMARY KEY,
		name TEXT UNIQUE NOT NULL
	);
	CREATE TABLE tags (
		id   INTEGER PRIMARY KEY,
		name TEXT UNIQUE NOT NULL
	);
	CREATE TABLE tasks (
		id           INTEGER PRIMARY KEY,
		title        TEXT NOT NULL,
		notes        TEXT NOT NULL DEFAULT '',
		project_id   INTEGER REFERENCES projects(id) ON DELETE SET NULL,
		priority     INTEGER NOT NULL DEFAULT 0,
		due          TEXT,
		created_at   TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
		completed_at TEXT
	);
	CREATE TABLE task_tags (
		task_id INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
		tag_id  INTEGER NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
		PRIMARY KEY (task_id, tag_id)
	);
	CREATE INDEX tasks_due ON tasks(due);`,
}

// DefaultPath returns $TD_DB, else $XDG_DATA_HOME/td/td.db, else ~/.local/share/td/td.db.
func DefaultPath() string {
	if p := os.Getenv("TD_DB"); p != "" {
		return p
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "td", "td.db")
}

// Open opens (creating if needed) the database at path and applies migrations.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	dsn := "file:" + path + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate %s: %w", path, err)
	}
	return s, nil
}

// Close releases the database.
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	var v int
	if err := s.db.QueryRow("PRAGMA user_version").Scan(&v); err != nil {
		return err
	}
	for i := v; i < len(migrations); i++ {
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(migrations[i]); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", i+1)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
