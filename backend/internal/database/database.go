package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id            TEXT PRIMARY KEY,
	email         TEXT NOT NULL UNIQUE,
	username      TEXT NOT NULL UNIQUE COLLATE NOCASE,
	display_name  TEXT NOT NULL,
	avatar        TEXT NOT NULL DEFAULT '',
	password_hash TEXT NOT NULL,
	created_at    TEXT NOT NULL,
	updated_at    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
	token      TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at TEXT NOT NULL,
	expires_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_user ON sessions(user_id);

CREATE TABLE IF NOT EXISTS projects (
	id         TEXT PRIMARY KEY,
	owner_id   TEXT NOT NULL DEFAULT '',
	name       TEXT NOT NULL,
	format     TEXT NOT NULL,
	notes      TEXT NOT NULL DEFAULT '[]',
	rig        TEXT NOT NULL,
	source     BLOB,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS projects_owner ON projects(owner_id, updated_at DESC);

CREATE TABLE IF NOT EXISTS animations (
	project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	name       TEXT NOT NULL,
	position   INTEGER NOT NULL,
	payload    TEXT NOT NULL,
	PRIMARY KEY (project_id, name)
);
`

func Open(dir string) (*sql.DB, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	path := filepath.Join(dir, "mcaia.db")

	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("database: open: %w", err)
	}

	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database: ping: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("database: migrate: %w", err)
	}

	if err := addMissingColumns(db); err != nil {
		return nil, err
	}

	return db, nil
}

var additions = map[string]map[string]string{
	"projects": {"source": "BLOB"},
}

func addMissingColumns(db *sql.DB) error {
	for table, columns := range additions {
		existing, err := columnNames(db, table)
		if err != nil {
			return err
		}

		for column, definition := range columns {
			if existing[column] {
				continue
			}

			statement := "ALTER TABLE " + table + " ADD COLUMN " + column + " " + definition

			if _, err := db.Exec(statement); err != nil {
				return fmt.Errorf("database: %s: %w", statement, err)
			}
		}
	}

	return nil
}

func columnNames(db *sql.DB, table string) (map[string]bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	names := make(map[string]bool)

	for rows.Next() {
		var (
			index      int
			name       string
			kind       string
			notNull    int
			defaults   any
			primaryKey int
		)

		if err := rows.Scan(&index, &name, &kind, &notNull, &defaults, &primaryKey); err != nil {
			return nil, err
		}

		names[name] = true
	}

	return names, rows.Err()
}
