package service

import (
	"context"

	engine "github.com/Haroon-BCBP/workflow_engine/internal/workflow"
)
func (s *workflowService) SendTransition(ctx context.Context, workflowID string, sig engine.TransitionSignal) error {
	return s.temporalClient.SignalWorkflow(ctx, workflowID, "", engine.TransitionChannel, sig)
}

func (s *workflowService) SendComment(ctx context.Context, workflowID string, sig engine.CommentSignal) error {
	return s.temporalClient.SignalWorkflow(ctx, workflowID, "", engine.CommentChannel, sig)
}

func (s *workflowService) SendAdminRouting(ctx context.Context, workflowID string, sig engine.AdminRoutingSignal) error {
	return s.temporalClient.SignalWorkflow(ctx, workflowID, "", engine.AdminRoutingChannel, sig)
}

func (s *workflowService) SendAdminStart(ctx context.Context, workflowID string, sig engine.AdminStartSignal) error {
	return s.temporalClient.SignalWorkflow(ctx, workflowID, "", engine.AdminStartChannel, sig)
}