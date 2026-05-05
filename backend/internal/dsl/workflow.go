package dsl

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

func DSLWorkflow(ctx workflow.Context, def WorkflowDef) (string, error) {
	logger := workflow.GetLogger(ctx)

	// Initialise runtime state
	state := &WorkflowState{
		WorkflowID:  workflow.GetInfo(ctx).WorkflowExecution.ID,
		Name:        def.Name,
		CurrentStep: 0,
		Progress:    make(map[string]*DepartmentProgress),
		Status:      WorkflowRunning,
	}
	for _, d := range def.Departments {
		state.Progress[d.ID] = &DepartmentProgress{
			DeptID:       d.ID,
			Label:        d.Label,
			CurrentStage: StagePrep,
			StageStatus:  StageStatusPending,
		}
	}

	if err := workflow.SetQueryHandler(ctx, QueryStatus, func() (*WorkflowState, error) {
		return state, nil
	}); err != nil {
		return "", fmt.Errorf("failed to set query handler: %w", err)
	}

	// shared across all department goroutines
	transitionChan := workflow.GetSignalChannel(ctx, TransitionChannel)
	commentChan := workflow.GetSignalChannel(ctx, CommentChannel)
	adminChan := workflow.GetSignalChannel(ctx, AdminRoutingChannel)

	for stepIdx, step := range def.Execution.Steps {
		state.CurrentStep = stepIdx

		if len(step.Sequential) > 0 {
			for _, deptID := range step.Sequential {
				dept := findDept(def, deptID)
				if dept == nil {
					return "", fmt.Errorf("dept %q not found in definition", deptID)
				}
				rejected, err := processDepartment(ctx, *dept, def, state, transitionChan, commentChan, adminChan)
				if err != nil {
					return "", err
				}
				if rejected {
					logger.Info("Workflow paused for admin routing", "dept", deptID)
					routed, err := waitForAdminRouting(ctx, def, state, transitionChan, commentChan, adminChan)
					if err != nil || !routed {
						state.Status = WorkflowRejected
						return "Workflow terminated by admin", nil
					}
					stepIdx = -1
					break
				}
			}
		}

		if len(step.Parallel) > 0 {
			if err := runParallel(ctx, step.Parallel, def, state, transitionChan, commentChan, adminChan); err != nil {
				return "", err
			}
		}
	}

	state.Status = WorkflowApproved
	logger.Info("Workflow fully approved", "name", def.Name)
	return "Workflow Fully Approved", nil
}

// processDepartment runs a single department through its stages (Prep → Review → Approve).
// Returns (rejected=true, nil) when the department is rejected and admin routing is needed.
func processDepartment(
	ctx workflow.Context,
	dept DepartmentDef,
	def WorkflowDef,
	state *WorkflowState,
	transitionChan, commentChan workflow.ReceiveChannel,
	adminChan workflow.ReceiveChannel,
) (rejected bool, err error) {
	logger := workflow.GetLogger(ctx)
	progress := state.Progress[dept.ID]

	for _, stage := range dept.Stages {
		progress.CurrentStage = stage.Type
		progress.StageStatus = StageStatusInProgress
		progress.HasComment = false

		logger.Info("Department stage started", "dept", dept.ID, "stage", stage.Type)

		ao := workflow.ActivityOptions{StartToCloseTimeout: 10 * time.Second}
		actCtx := workflow.WithActivityOptions(ctx, ao)
		_ = workflow.ExecuteActivity(actCtx, StageStartedActivity, dept.ID, string(stage.Type)).Get(actCtx, nil)

		// Wait for user signals (transition or comment)
		for {
			var done bool
			var wasRejected bool
			selector := workflow.NewSelector(ctx)

			selector.AddReceive(transitionChan, func(c workflow.ReceiveChannel, _ bool) {
				var sig TransitionSignal
				c.Receive(ctx, &sig)

				if sig.DeptID != dept.ID {
					// For POC: log and skip (signal ordering is handled by admin/UI sequencing)
					logger.Warn("Signal for different dept, skipping", "for", sig.DeptID, "current", dept.ID)
					return
				}

				if sig.ToStage == StageApprove {
					// Approve only valid at the approve stage
					if stage.Type != StageApprove {
						logger.Warn("Cannot approve at this stage", "stage", stage.Type)
						return
					}
					if stage.RequiresComment && !progress.HasComment {
						logger.Warn("Cannot approve without a comment", "dept", dept.ID)
						return
					}
					progress.StageStatus = StageStatusDone
					done = true
				} else if sig.ToStage == StageReview {
					if stage.Type != StagePrep {
						logger.Warn("Unexpected review transition", "stage", stage.Type)
						return
					}
					progress.StageStatus = StageStatusDone
					done = true
				} else if sig.ToStage == "reject" {
					progress.StageStatus = StageStatusRejected
					state.RejectedBy = dept.ID
					state.Status = WorkflowPaused
					done = true
					wasRejected = true
				}
			})

			selector.AddReceive(commentChan, func(c workflow.ReceiveChannel, _ bool) {
				var sig CommentSignal
				c.Receive(ctx, &sig)
				if sig.DeptID != dept.ID {
					return
				}
				_ = workflow.ExecuteActivity(
					workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 10 * time.Second}),
					SaveCommentActivity, sig.DeptID, string(sig.Stage), sig.UserID, sig.Text,
				).Get(ctx, nil)
				progress.HasComment = true
				progress.Comments = append(progress.Comments, Comment{
					UserID: sig.UserID,
					Text:   sig.Text,
					Stage:  sig.Stage,
				})
			})

			selector.Select(ctx)

			if done {
				if wasRejected {
					return true, nil
				}
				break
			}
		}
	}

	progress.StageStatus = StageStatusDone
	return false, nil
}

// waitForAdminRouting blocks until admin sends a routing or terminate signal.
// Returns true if admin chose to route (continue), false if terminate.
func waitForAdminRouting(
	ctx workflow.Context,
	def WorkflowDef,
	state *WorkflowState,
	transitionChan, commentChan, adminChan workflow.ReceiveChannel,
) (bool, error) {
	logger := workflow.GetLogger(ctx)
	var routed bool

	selector := workflow.NewSelector(ctx)
	selector.AddReceive(adminChan, func(c workflow.ReceiveChannel, _ bool) {
		var sig AdminRoutingSignal
		c.Receive(ctx, &sig)

		if sig.Action == "terminate" {
			logger.Info("Admin terminated workflow", "admin", sig.AdminID)
			routed = false
			return
		}

		if sig.Action == "goto" {
			logger.Info("Admin routing workflow", "admin", sig.AdminID, "dept", sig.DeptID, "stage", sig.Stage)
			// Reset progress for the target dept and all subsequent depts
			resetFrom(def, state, sig.DeptID, sig.Stage)
			routed = true
		}
	})
	selector.Select(ctx)
	return routed, nil
}

// runParallel runs a set of departments concurrently and waits for all to complete.
func runParallel(
	ctx workflow.Context,
	deptIDs []string,
	def WorkflowDef,
	state *WorkflowState,
	transitionChan, commentChan, adminChan workflow.ReceiveChannel,
) error {
	childCtx, cancel := workflow.WithCancel(ctx)
	selector := workflow.NewSelector(ctx)
	var firstErr error

	for _, deptID := range deptIDs {
		dept := findDept(def, deptID)
		if dept == nil {
			cancel()
			return fmt.Errorf("dept %q not found", deptID)
		}
		d := *dept
		future, settable := workflow.NewFuture(childCtx)
		workflow.Go(childCtx, func(ctx workflow.Context) {
			_, err := processDepartment(ctx, d, def, state, transitionChan, commentChan, adminChan)
			settable.Set(nil, err)
		})
		selector.AddFuture(future, func(f workflow.Future) {
			if err := f.Get(ctx, nil); err != nil && firstErr == nil {
				cancel()
				firstErr = err
			}
		})
	}

	for i := 0; i < len(deptIDs); i++ {
		selector.Select(ctx)
		if firstErr != nil {
			cancel()
			return firstErr
		}
	}
	cancel()
	return nil
}

// resetFrom clears progress for deptID (from given stage) and all depts after it in the plan.
func resetFrom(def WorkflowDef, state *WorkflowState, fromDeptID string, fromStage StageType) {
	found := false
	for _, step := range def.Execution.Steps {
		depts := append(step.Sequential, step.Parallel...)
		for _, id := range depts {
			if id == fromDeptID {
				found = true
			}
			if found {
				if p, ok := state.Progress[id]; ok {
					p.CurrentStage = fromStage
					p.StageStatus = StageStatusPending
					p.HasComment = false
					p.Comments = nil
				}
			}
		}
	}
	state.Status = WorkflowRunning
	state.RejectedBy = ""
}

func findDept(def WorkflowDef, id string) *DepartmentDef {
	for i, d := range def.Departments {
		if d.ID == id {
			return &def.Departments[i]
		}
	}
	return nil
}
