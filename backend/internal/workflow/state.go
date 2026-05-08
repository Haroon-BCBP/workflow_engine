package workflow

import "go.temporal.io/sdk/workflow"


func (p *DepartmentProgress) IsActive() bool {
	return p.StageStatus == StageStatusInProgress || p.StageStatus == StageStatusPending
}

func (p *DepartmentProgress) AllAssignees() []string {
	var ids []string
	for _, list := range p.StageAssignees {
		for _, id := range list {
			if id != "" {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

func (s *WorkflowState) UpdateWorkload() {
	w := make(map[string]int)
	if s.Status != WorkflowRunning && s.Status != WorkflowPaused {
		s.Workload = w
		return
	}

	for _, p := range s.Progress {
		if p.IsActive() {
			for _, id := range p.AllAssignees() {
				w[id]++
			}
		}
	}
	s.Workload = w
}

func (s *WorkflowState) ApplyAssignments(assignments map[string]map[StageType][]Assignment) {
	for deptID, stageAssignments := range assignments {
		if progress, ok := s.Progress[deptID]; ok {
			for stage, assignments := range stageAssignments {
				for _, assignment := range assignments {
					progress.StageAssignees[stage] = append(progress.StageAssignees[stage], assignment.UserID)
					progress.StageAssigneeNames[stage] = append(progress.StageAssigneeNames[stage], assignment.UserName)
				}
			}
		}
	}
}

func initState(ctx workflow.Context, def WorkflowDef) *WorkflowState {
	state := &WorkflowState{
		WorkflowID:  workflow.GetInfo(ctx).WorkflowExecution.ID,
		Name:        def.Name,
		CurrentStep: 0,
		Progress:    make(map[string]*DepartmentProgress),
		Execution:   def.Execution,
		Status:      WorkflowPendingAssignment,
	}
	for _, d := range def.Departments {
		state.Progress[d.ID] = &DepartmentProgress{
			DeptID:             d.ID,
			Label:              d.Label,
			CurrentStage:       StagePrep,
			StageStatus:        StageStatusPending,
			StageAssignees:     make(map[StageType][]string),
			StageAssigneeNames: make(map[StageType][]string),
		}
	}
	return state
}

func resetFrom(def WorkflowDef, state *WorkflowState, fromDeptID string, fromStage StageType) {
	found := false
	targetStepIdx := -1
	for i, step := range def.Execution.Steps {
		depts := append(step.Sequential, step.Parallel...)
		depts = append(depts, step.Exclusive...)

		if contains(depts, fromDeptID) {
			found = true
			if targetStepIdx == -1 {
				targetStepIdx = i
			}
		}

		if found {
			resetStepProgress(def, state, depts, fromDeptID, fromStage)
		}
	}

	if targetStepIdx != -1 {
		state.CurrentStep = targetStepIdx
	}
	state.Status = WorkflowRunning
	state.UpdateWorkload()
	state.RejectedBy = ""
}

func resetStepProgress(def WorkflowDef, state *WorkflowState, deptIDs []string, fromDeptID string, fromStage StageType) {
	for _, id := range deptIDs {
		if p, ok := state.Progress[id]; ok {
			if id == fromDeptID {
				p.CurrentStage = fromStage
			} else {
				deptDef := findDept(def, id)
				if deptDef != nil && len(deptDef.Stages) > 0 {
					p.CurrentStage = deptDef.Stages[0].Type
				}
			}
			p.StageStatus = StageStatusPending
			p.HasComment = false
			p.Comments = nil
		}
	}
}


