package repository

import (
	"time"
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

type Document struct {
	ID         string    `json:"id"`
	WorkflowID string    `json:"workflow_id"`
	DeptID     string    `json:"dept_id"`
	Stage      string    `json:"stage"`
	Filename   string    `json:"filename"`
	UserID     string    `json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
}

type SubmitResult struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
	YAML       string `json:"yaml"`
}

type RunSummary struct {
	WorkflowRun
	Status string `json:"status"`
}
