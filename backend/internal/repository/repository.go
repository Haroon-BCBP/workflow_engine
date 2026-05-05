package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type WorkflowRun struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	BPMNXML    string    `json:"bpmn_xml"`
	DSLYAML    string    `json:"dsl_yaml"`
	TemporalID string    `json:"temporal_id"`
	RunID      string    `json:"run_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type Repository struct {
	db *sql.DB
}

func New(dsn string) (*Repository, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("repository: open db: %w", err)
	}
	r := &Repository{db: db}
	if err := r.migrate(); err != nil {
		return nil, err
	}
	return r, nil
}

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
	`)
	return err
}

func (r *Repository) Save(ctx context.Context, run WorkflowRun) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO workflow_runs (id, name, bpmn_xml, dsl_yaml, temporal_id, run_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		run.ID, run.Name, run.BPMNXML, run.DSLYAML, run.TemporalID, run.RunID, run.CreatedAt,
	)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id string) (*WorkflowRun, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, bpmn_xml, dsl_yaml, temporal_id, run_id, created_at FROM workflow_runs WHERE id = ?`, id,
	)
	var run WorkflowRun
	if err := row.Scan(&run.ID, &run.Name, &run.BPMNXML, &run.DSLYAML, &run.TemporalID, &run.RunID, &run.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("repository: workflow %q not found", id)
		}
		return nil, err
	}
	return &run, nil
}

func (r *Repository) List(ctx context.Context) ([]WorkflowRun, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, bpmn_xml, dsl_yaml, temporal_id, run_id, created_at FROM workflow_runs ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var runs []WorkflowRun
	for rows.Next() {
		var run WorkflowRun
		if err := rows.Scan(&run.ID, &run.Name, &run.BPMNXML, &run.DSLYAML, &run.TemporalID, &run.RunID, &run.CreatedAt); err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, rows.Err()
}
