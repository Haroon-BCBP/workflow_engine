package service

import (
	"context"

	"go.temporal.io/sdk/client"

	"github.com/Haroon-BCBP/workflow_engine/internal/bpmn"
	engine "github.com/Haroon-BCBP/workflow_engine/internal/workflow"
	"github.com/Haroon-BCBP/workflow_engine/internal/iam"
	"github.com/Haroon-BCBP/workflow_engine/internal/repository"
)

type WorkflowService interface {
	Submit(ctx context.Context, bpmnXML string) (*repository.SubmitResult, error)
	GetStatus(ctx context.Context, workflowID string) (*engine.WorkflowState, error)
	SendTransition(ctx context.Context, workflowID string, sig engine.TransitionSignal) error
	SendComment(ctx context.Context, workflowID string, sig engine.CommentSignal) error
	SendAdminRouting(ctx context.Context, workflowID string, sig engine.AdminRoutingSignal) error
	SendAdminStart(ctx context.Context, workflowID string, sig engine.AdminStartSignal) error
	GetWorkloads(ctx context.Context) (map[string]int, error)
	ListRuns(ctx context.Context, userID string, isAdmin bool) ([]repository.RunSummary, error)
	UploadDocument(ctx context.Context, workflowID, deptID, stage, filename, userID string) (repository.Document, error)
	GetDocuments(ctx context.Context, workflowID, deptID, stage string) ([]repository.Document, error)
	GetYAML(ctx context.Context, workflowID string) (string, error)
}


func New(repo *repository.Repository, tc client.Client, i *iam.IAM) WorkflowService {
	return &workflowService{
		repo:           repo,
		temporalClient: tc,
		parser:         bpmn.NewParser(nil),
		iam:            i,
	}
}
