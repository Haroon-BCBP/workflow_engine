package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"gopkg.in/yaml.v3"

	"github.com/Haroon-BCBP/workflow_engine/internal/bpmn"
	"github.com/Haroon-BCBP/workflow_engine/internal/dsl"
	"github.com/Haroon-BCBP/workflow_engine/internal/repository"
)

type SubmitResult struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
	YAML       string `json:"yaml"`
}

type WorkflowService struct {
	repo           *repository.Repository
	temporalClient client.Client
	parser         *bpmn.Parser
}

func New(repo *repository.Repository, tc client.Client) *WorkflowService {
	return &WorkflowService{
		repo:           repo,
		temporalClient: tc,
		parser:         &bpmn.Parser{},
	}
}

func (s *WorkflowService) Submit(ctx context.Context, bpmnXML string) (*SubmitResult, error) {
	def, err := s.parser.ParseXML([]byte(bpmnXML))
	if err != nil {
		return nil, fmt.Errorf("service: parse bpmn: %w", err)
	}

	yamlBytes, err := yaml.Marshal(def)
	if err != nil {
		return nil, fmt.Errorf("service: marshal yaml: %w", err)
	}
	yamlStr := string(yamlBytes)

	workflowID := "wf-" + uuid.New().String()
	opts := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: dsl.TaskQueue,
	}
	we, err := s.temporalClient.ExecuteWorkflow(ctx, opts, dsl.DSLWorkflow, *def)
	if err != nil {
		return nil, fmt.Errorf("service: start temporal workflow: %w", err)
	}

	run := repository.WorkflowRun{
		ID:         workflowID,
		Name:       def.Name,
		BPMNXML:    bpmnXML,
		DSLYAML:    yamlStr,
		TemporalID: we.GetID(),
		RunID:      we.GetRunID(),
		CreatedAt:  time.Now(),
	}
	if err := s.repo.Save(ctx, run); err != nil {
		return nil, fmt.Errorf("service: save run: %w", err)
	}

	return &SubmitResult{
		WorkflowID: workflowID,
		RunID:      we.GetRunID(),
		YAML:       yamlStr,
	}, nil
}

func (s *WorkflowService) GetStatus(ctx context.Context, workflowID string) (*dsl.WorkflowState, error) {
	// QueryRejectCondition_NONE allows querying closed (completed/terminated) workflows.
	// Without this, Temporal returns an error for non-running executions, leaving the UI
	// stuck on the last known state instead of reflecting the final rejected/approved status.
	resp, err := s.temporalClient.QueryWorkflowWithOptions(ctx, &client.QueryWorkflowWithOptionsRequest{
		WorkflowID:           workflowID,
		QueryType:            dsl.QueryStatus,
		QueryRejectCondition: enums.QUERY_REJECT_CONDITION_NONE,
	})
	if err != nil {
		return nil, fmt.Errorf("service: query workflow: %w", err)
	}
	var state dsl.WorkflowState
	if err := resp.QueryResult.Get(&state); err != nil {
		return nil, fmt.Errorf("service: decode state: %w", err)
	}

	if len(state.Execution.Steps) == 0 {
		run, err := s.repo.GetByID(ctx, workflowID)
		if err == nil {
			var def dsl.WorkflowDef
			if err := yaml.Unmarshal([]byte(run.DSLYAML), &def); err == nil {
				state.Execution = def.Execution
			}
		}
	}

	return &state, nil
}

func (s *WorkflowService) SendTransition(ctx context.Context, workflowID string, sig dsl.TransitionSignal) error {
	return s.temporalClient.SignalWorkflow(ctx, workflowID, "", dsl.TransitionChannel, sig)
}

func (s *WorkflowService) SendComment(ctx context.Context, workflowID string, sig dsl.CommentSignal) error {
	return s.temporalClient.SignalWorkflow(ctx, workflowID, "", dsl.CommentChannel, sig)
}

func (s *WorkflowService) SendAdminRouting(ctx context.Context, workflowID string, sig dsl.AdminRoutingSignal) error {
	return s.temporalClient.SignalWorkflow(ctx, workflowID, "", dsl.AdminRoutingChannel, sig)
}

func (s *WorkflowService) ListRuns(ctx context.Context) ([]repository.WorkflowRun, error) {
	return s.repo.List(ctx)
}

func (s *WorkflowService) UploadDocument(ctx context.Context, workflowID, deptID, stage, filename, userID string) (repository.Document, error) {
	doc := repository.Document{
		ID:         uuid.New().String(),
		WorkflowID: workflowID,
		DeptID:     deptID,
		Stage:      stage,
		Filename:   filename,
		UserID:     userID,
		CreatedAt:  time.Now(),
	}
	if err := s.repo.SaveDocument(ctx, doc); err != nil {
		return doc, err
	}
	err := s.temporalClient.SignalWorkflow(ctx, workflowID, "", dsl.DocumentChannel, dsl.DocumentSignal{
		DeptID: deptID,
		Stage:  dsl.StageType(stage),
	})
	return doc, err
}

func (s *WorkflowService) GetDocuments(ctx context.Context, workflowID, deptID, stage string) ([]repository.Document, error) {
	return s.repo.GetDocuments(ctx, workflowID, deptID, stage)
}

func (s *WorkflowService) GetYAML(ctx context.Context, workflowID string) (string, error) {
	run, err := s.repo.GetByID(ctx, workflowID)
	if err != nil {
		return "", err
	}
	return run.DSLYAML, nil
}
