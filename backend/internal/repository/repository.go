package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type Repository struct {
	db *DB
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

func (r *Repository) SaveDocument(ctx context.Context, doc Document) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO documents (id, workflow_id, dept_id, stage, filename, user_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		doc.ID, doc.WorkflowID, doc.DeptID, doc.Stage, doc.Filename, doc.UserID, doc.CreatedAt,
	)
	return err
}

func (r *Repository) GetDocuments(ctx context.Context, workflowID, deptID, stage string) ([]Document, error) {
	query := `SELECT id, workflow_id, dept_id, stage, filename, user_id, created_at FROM documents WHERE workflow_id = ?`
	args := []any{workflowID}
	if deptID != "" {
		query += ` AND dept_id = ?`
		args = append(args, deptID)
	}
	if stage != "" {
		query += ` AND stage = ?`
		args = append(args, stage)
	}
	query += ` ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var docs []Document
	for rows.Next() {
		var doc Document
		if err := rows.Scan(&doc.ID, &doc.WorkflowID, &doc.DeptID, &doc.Stage, &doc.Filename, &doc.UserID, &doc.CreatedAt); err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, rows.Err()
}

