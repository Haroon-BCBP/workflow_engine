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
	engine "github.com/Haroon-BCBP/workflow_engine/internal/workflow"
	"github.com/Haroon-BCBP/workflow_engine/internal/iam"
	"github.com/Haroon-BCBP/workflow_engine/internal/repository"
)

type workflowService struct {
	repo           *repository.Repository
	temporalClient client.Client
	parser         *bpmn.Parser
	iam            *iam.IAM
}

func (s *workflowService) Submit(ctx context.Context, bpmnXML string) (*repository.SubmitResult, error) {
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
		TaskQueue: engine.TaskQueue,
	}
	we, err := s.temporalClient.ExecuteWorkflow(ctx, opts, engine.DSLWorkflow, *def)
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

	return &repository.SubmitResult{
		WorkflowID: workflowID,
		RunID:      we.GetRunID(),
		YAML:       yamlStr,
	}, nil
}

func (s *workflowService) GetStatus(ctx context.Context, workflowID string) (*engine.WorkflowState, error) {
	resp, err := s.temporalClient.QueryWorkflowWithOptions(ctx, &client.QueryWorkflowWithOptionsRequest{
		WorkflowID:           workflowID,
		QueryType:            engine.QueryStatus,
		QueryRejectCondition: enums.QUERY_REJECT_CONDITION_NONE,
	})
	if err != nil {
		return nil, fmt.Errorf("service: query workflow: %w", err)
	}
	var state engine.WorkflowState
	if err := resp.QueryResult.Get(&state); err != nil {
		return nil, fmt.Errorf("service: decode state: %w", err)
	}

	if len(state.Execution.Steps) == 0 {
		run, err := s.repo.GetByID(ctx, workflowID)
		if err == nil {
			var def engine.WorkflowDef
			if err := yaml.Unmarshal([]byte(run.DSLYAML), &def); err == nil {
				state.Execution = def.Execution
			}
		}
	}

	return &state, nil
}

func (s *workflowService) GetWorkloads(ctx context.Context) (map[string]int, error) {
	runs, err := s.ListRuns(ctx, "", true)
	if err != nil {
		return nil, err
	}

	type runResult struct {
		workloads map[string]int
	}
	resChan := make(chan runResult, len(runs))

	for _, run := range runs {
		go func(r repository.WorkflowRun) {
			resChan <- runResult{workloads: s.calculateRunWorkload(ctx, r.ID)}
		}(run.WorkflowRun)
	}

	totalWorkloads := make(map[string]int)
	for i := 0; i < len(runs); i++ {
		res := <-resChan
		for id, count := range res.workloads {
			totalWorkloads[id] += count
		}
	}
	return totalWorkloads, nil
}

func (s *workflowService) ListRuns(ctx context.Context, userID string, isAdmin bool) ([]repository.RunSummary, error) {
	runs, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	type runResult struct {
		summary    repository.RunSummary
		isAssigned bool
	}
	resChan := make(chan runResult, len(runs))

	for _, run := range runs {
		go func(r repository.WorkflowRun) {
			summary, isAssigned := s.summarizeRun(ctx, r, userID, isAdmin)
			resChan <- runResult{summary: summary, isAssigned: isAssigned}
		}(run)
	}

	var filtered []repository.RunSummary
	for i := 0; i < len(runs); i++ {
		res := <-resChan
		if res.isAssigned {
			filtered = append(filtered, res.summary)
		}
	}
	return filtered, nil
}

func (s *workflowService) UploadDocument(ctx context.Context, workflowID, deptID, stage, filename, userID string) (repository.Document, error) {
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
	err := s.temporalClient.SignalWorkflow(ctx, workflowID, "", engine.DocumentChannel, engine.DocumentSignal{
		DeptID: deptID,
		Stage:  engine.StageType(stage),
	})
	return doc, err
}

func (s *workflowService) GetDocuments(ctx context.Context, workflowID, deptID, stage string) ([]repository.Document, error) {
	return s.repo.GetDocuments(ctx, workflowID, deptID, stage)
}

func (s *workflowService) GetYAML(ctx context.Context, workflowID string) (string, error) {
	run, err := s.repo.GetByID(ctx, workflowID)
	if err != nil {
		return "", err
	}
	return run.DSLYAML, nil
}
