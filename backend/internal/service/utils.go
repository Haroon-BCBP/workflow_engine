package service

import (
	"context"
	"time"

	engine "github.com/Haroon-BCBP/workflow_engine/internal/workflow"
	"github.com/Haroon-BCBP/workflow_engine/internal/repository"
)



func (s *workflowService) calculateRunWorkload(ctx context.Context, workflowID string) map[string]int {
	queryCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	state, err := s.GetStatus(queryCtx, workflowID)
	if err != nil {
		return nil
	}

	w := make(map[string]int)
	if state.Status != engine.WorkflowRunning && state.Status != engine.WorkflowPaused {
		return nil
	}

	if state.Workload != nil {
		return state.Workload
	}

	return w
}

func (s *workflowService) summarizeRun(ctx context.Context, r repository.WorkflowRun, userID string, isAdmin bool) (repository.RunSummary, bool) {
	queryCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	state, err := s.GetStatus(queryCtx, r.ID)
	status := "unknown"
	if err == nil {
		status = string(state.Status)
	}

	summary := repository.RunSummary{
		WorkflowRun: r,
		Status:      status,
	}

	isAssigned := isAdmin || userID == ""
	if !isAssigned && err == nil {
		isAssigned = s.isUserAssignedToRun(userID, state)
	}

	return summary, isAssigned
}

func (s *workflowService) isUserAssignedToRun(userID string, state *engine.WorkflowState) bool {
	userDepts := s.iam.GetUserDepartments(userID)
	deptMap := make(map[string]struct{})
	for _, d := range userDepts {
		deptMap[d] = struct{}{}
	}

	for _, progress := range state.Progress {
		if _, ok := deptMap[progress.DeptID]; ok {
			return true
		}
	}
	return false
}
