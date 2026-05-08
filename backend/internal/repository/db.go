package repository

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func Connect(dsn string) (*DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db: ping: %w", err)
	}

	return &DB{db}, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}


func New(db *DB) (*Repository, error) {
	r := &Repository{db: db}
	if err := r.migrate(); err != nil {
		return nil, err
	}
	return r, nil
}

// POC, setup schema using goose/sqlc later
func (r *Repository) migrate() error {
	_, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS workflow_runs (
			id          TEXT PRIMARY KEY,
			name        TEXT NOT NULL,
			bpmn_xml    TEXT NOT NULL,
			dsl_yaml    TEXT NOT NULL,
			temporal_id TEXT NOT NULL,
			run_id      TEXT NOT NULL,
			created_at  DATETIME NOT NULL
		);
		CREATE TABLE IF NOT EXISTS documents (
			id          TEXT PRIMARY KEY,
			workflow_id TEXT NOT NULL,
			dept_id     TEXT NOT NULL,
			stage       TEXT NOT NULL,
			filename    TEXT NOT NULL,
			user_id     TEXT NOT NULL,
			created_at  DATETIME NOT NULL,
			FOREIGN KEY(workflow_id) REFERENCES workflow_runs(id) ON DELETE CASCADE
		);
	`)
	return err
}