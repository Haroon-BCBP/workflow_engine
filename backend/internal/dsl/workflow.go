package dsl

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
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

	if err := workflow.SetQueryHandler(ctx, QueryStatus, func() (*WorkflowState, error) {
		return state, nil
	}); err != nil {
		return "", fmt.Errorf("failed to set query handler: %w", err)
	}

	transitionChan := workflow.GetSignalChannel(ctx, TransitionChannel)
	commentChan := workflow.GetSignalChannel(ctx, CommentChannel)

	deptTransitionChans := make(map[string]workflow.Channel)
	deptCommentChans := make(map[string]workflow.Channel)
	for _, d := range def.Departments {
		deptTransitionChans[d.ID] = workflow.NewBufferedChannel(ctx, 1024)
		deptCommentChans[d.ID] = workflow.NewBufferedChannel(ctx, 1024)
	}

	workflow.Go(ctx, func(ctx workflow.Context) {
		for {
			selector := workflow.NewSelector(ctx)
			selector.AddReceive(transitionChan, func(c workflow.ReceiveChannel, _ bool) {
				var sig TransitionSignal
				c.Receive(ctx, &sig)
				if ch, ok := deptTransitionChans[sig.DeptID]; ok {
					workflow.Go(ctx, func(ctx workflow.Context) {
						ch.Send(ctx, sig)
					})
				} else {
					logger.Warn("Received transition signal for unknown dept", "dept", sig.DeptID)
				}
			})
			selector.AddReceive(commentChan, func(c workflow.ReceiveChannel, _ bool) {
				var sig CommentSignal
				c.Receive(ctx, &sig)
				if ch, ok := deptCommentChans[sig.DeptID]; ok {
					workflow.Go(ctx, func(ctx workflow.Context) {
						ch.Send(ctx, sig)
					})
				} else {
					logger.Warn("Received comment signal for unknown dept", "dept", sig.DeptID)
				}
			})
			selector.Select(ctx)
			if ctx.Err() != nil {
				return
			}
		}
	})

	adminStartChan := workflow.GetSignalChannel(ctx, AdminStartChannel)
	adminChan := workflow.GetSignalChannel(ctx, AdminRoutingChannel)


	logger.Info("Workflow pending admin assignment", "name", def.Name)
	var startSig AdminStartSignal
	adminStartChan.Receive(ctx, &startSig)
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	logger.Info("Admin assigned identities and started workflow", "admin", startSig.AdminID)
	for deptID, stageAssignments := range startSig.Assignments {
		if progress, ok := state.Progress[deptID]; ok {
			for stage, assignments := range stageAssignments {
				for _, assignment := range assignments {
					progress.StageAssignees[stage] = append(progress.StageAssignees[stage], assignment.UserID)
					progress.StageAssigneeNames[stage] = append(progress.StageAssigneeNames[stage], assignment.UserName)
				}
			}
		}
	}
	state.Status = WorkflowRunning

OuterLoop:
	for {
		for stepIdx := state.CurrentStep; stepIdx < len(def.Execution.Steps); stepIdx++ {
			step := def.Execution.Steps[stepIdx]
			state.CurrentStep = stepIdx

			if len(step.Sequential) > 0 {
				for _, deptID := range step.Sequential {
					dept := findDept(def, deptID)
					if dept == nil {
						return "", fmt.Errorf("dept %q not found in definition", deptID)
					}
					rejected, err := processDepartment(ctx, *dept, def, state, deptTransitionChans[deptID], deptCommentChans[deptID])
					if err != nil {
						return "", err
					}
					if rejected {
						logger.Info("Workflow paused for admin routing", "dept", deptID)
						routed, err := waitForAdminRouting(ctx, def, state, adminChan)
						if err != nil || !routed {
							state.Status = WorkflowRejected
							return "Workflow terminated by admin", nil
						}
						continue OuterLoop
					}
				}
			}

			if len(step.Parallel) > 0 {
				rejected, err := runParallel(ctx, step.Parallel, def, state, deptTransitionChans, deptCommentChans)
				if err != nil {
					return "", err
				}
				if rejected {
					logger.Info("Workflow paused for admin routing (parallel rejection)")
					routed, err := waitForAdminRouting(ctx, def, state, adminChan)
					if err != nil || !routed {
						state.Status = WorkflowRejected
						return "Workflow terminated by admin", nil
					}
					continue OuterLoop
				}
			}

			if len(step.Exclusive) > 0 {
				logger.Info("Workflow paused for XOR routing", "options", step.Exclusive)
				state.Status = WorkflowPausedXOR
				
				var chosenDept string
				for {
					var sig AdminRoutingSignal
					adminChan.Receive(ctx, &sig)
					if ctx.Err() != nil {
						return "", ctx.Err()
					}
					if sig.Action == "xor_route" {
						chosenDept = sig.DeptID
						break
					} else if sig.Action == "terminate" {
						state.Status = WorkflowRejected
						return "Workflow terminated by admin", nil
					}
				}
				
				state.Status = WorkflowRunning
				logger.Info("Admin chose XOR branch", "dept", chosenDept)
				
				dept := findDept(def, chosenDept)
				if dept == nil {
					return "", fmt.Errorf("dept %q not found in definition", chosenDept)
				}
				rejected, err := processDepartment(ctx, *dept, def, state, deptTransitionChans[chosenDept], deptCommentChans[chosenDept])
				if err != nil {
					return "", err
				}
				if rejected {
					logger.Info("Workflow paused for admin routing (xor rejection)", "dept", chosenDept)
					routed, err := waitForAdminRouting(ctx, def, state, adminChan)
					if err != nil || !routed {
						state.Status = WorkflowRejected
						return "Workflow terminated by admin", nil
					}
					continue OuterLoop
				}
			}
		}
		break // Finished all steps
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
) (rejected bool, err error) {
	logger := workflow.GetLogger(ctx)
	progress := state.Progress[dept.ID]

	stageIdx := 0
	for stageIdx < len(dept.Stages) {
		stage := dept.Stages[stageIdx]

		if progress.StageStatus == StageStatusPending && stage.Type != progress.CurrentStage {
			stageIdx++
			continue
		}

		progress.CurrentStage = stage.Type
		progress.StageStatus = StageStatusInProgress
		progress.HasComment = false

		logger.Info("Department stage started", "dept", dept.ID, "stage", stage.Type)

		ao := workflow.ActivityOptions{StartToCloseTimeout: 10 * time.Second}
		actCtx := workflow.WithActivityOptions(ctx, ao)
		_ = workflow.ExecuteActivity(actCtx, StageStartedActivity, dept.ID, string(stage.Type)).Get(actCtx, nil)

		var backToPrep bool
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

				switch sig.ToStage {
				case StageApprove:
					switch stage.Type {
					case StageReview:
						if progress.HasComment {
							logger.Info("Comments found during review, routing back to prep", "dept", dept.ID)
							backToPrep = true
							done = true
						} else {
							progress.StageStatus = StageStatusDone
							done = true
						}
					case StageApprove:
						if stage.RequiresComment && !progress.HasComment {
							logger.Warn("Cannot approve without a comment", "dept", dept.ID)
							return
						}
						progress.StageStatus = StageStatusDone
						done = true
					default:
						logger.Warn("Cannot approve/advance to approve from this stage", "stage", stage.Type)
					}
				case StageReview:
					if stage.Type != StagePrep {
						logger.Warn("Unexpected review transition", "stage", stage.Type)
						return
					}
					progress.StageStatus = StageStatusDone
					done = true
				case "reject":
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
			if ctx.Err() != nil {
				return false, ctx.Err()
			}

			if done {
				if wasRejected {
					return true, nil
				}
				break
			}
		}

		if backToPrep {
			prepIdx := 0
			for i, s := range dept.Stages {
				if s.Type == StagePrep {
					prepIdx = i
					break
				}
			}
			progress.StageStatus = StageStatusPending
			progress.CurrentStage = StagePrep
			stageIdx = prepIdx
		} else {
			stageIdx++
		}
	}

	if progress.CurrentStage == StageApprove && progress.StageStatus == StageStatusDone {
		return false, nil
	}
	return false, nil // Standard completion
}

// waitForAdminRouting blocks until admin sends a valid routing or terminate signal.
// Returns true if admin chose to route (continue), false if terminate.
// Loops on unrecognized actions so stale or malformed signals don't accidentally terminate.
func waitForAdminRouting(
	ctx workflow.Context,
	def WorkflowDef,
	state *WorkflowState,
	adminChan workflow.ReceiveChannel,
) (bool, error) {
	logger := workflow.GetLogger(ctx)

	for {
		var sig AdminRoutingSignal
		adminChan.Receive(ctx, &sig)
		if ctx.Err() != nil {
			return false, ctx.Err()
		}

		switch sig.Action {
		case "terminate":
			logger.Info("Admin terminated workflow", "admin", sig.AdminID)
			return false, nil
		case "goto":
			logger.Info("Admin routing workflow", "admin", sig.AdminID, "dept", sig.DeptID, "stage", sig.Stage)
			resetFrom(def, state, sig.DeptID, sig.Stage)
			return true, nil
		default:
			logger.Warn("Ignoring unrecognized admin action, waiting for next signal", "action", sig.Action)
		}
	}
}

// runParallel runs a set of departments concurrently and waits for all to complete.
func runParallel(
	ctx workflow.Context,
	deptIDs []string,
	def WorkflowDef,
	state *WorkflowState,
	deptTransitionChans, deptCommentChans map[string]workflow.Channel,
) (bool, error) {
	childCtx, cancel := workflow.WithCancel(ctx)
	defer cancel()

	selector := workflow.NewSelector(ctx)
	var firstErr error
	var wasRejected bool
	completedCount := 0

	for _, deptID := range deptIDs {
		dept := findDept(def, deptID)
		if dept == nil {
			return false, fmt.Errorf("dept %q not found", deptID)
		}
		d := *dept
		tChan := deptTransitionChans[deptID]
		cChan := deptCommentChans[deptID]

		future, settable := workflow.NewFuture(childCtx)
		workflow.Go(childCtx, func(ctx workflow.Context) {
			rejected, err := processDepartment(ctx, d, def, state, tChan, cChan)
			settable.Set(rejected, err)
		})
		selector.AddFuture(future, func(f workflow.Future) {
			var rejected bool
			if err := f.Get(ctx, &rejected); err != nil {
				if firstErr == nil && !temporal.IsCanceledError(err) {
					firstErr = err
				}
			} else if rejected {
				wasRejected = true
			}
			completedCount++
		})
	}

	for completedCount < len(deptIDs) {
		selector.Select(ctx)
		if firstErr != nil {
			return false, firstErr
		}
		if wasRejected {
			// Cancel sibling branches and drain their futures so Temporal's
			// goroutine scheduler is clean before we return.
			cancel()
			timerExpired := false
			timerFuture := workflow.NewTimer(ctx, 2*time.Second)
			selector.AddFuture(timerFuture, func(f workflow.Future) {
				timerExpired = true
			})
			for completedCount < len(deptIDs) && !timerExpired {
				selector.Select(ctx)
			}
			return true, nil
		}
	}

	return false, nil
}

// resetFrom clears progress for deptID (from given stage) and all depts after it in the plan.
func resetFrom(def WorkflowDef, state *WorkflowState, fromDeptID string, fromStage StageType) {
	found := false
	targetStepIdx := -1
	for i, step := range def.Execution.Steps {
		// Check if this step contains our target department
		depts := append(step.Sequential, step.Parallel...)
		stepContainsTarget := false
		for _, id := range depts {
			if id == fromDeptID {
				stepContainsTarget = true
				break
			}
		}

		if stepContainsTarget {
			found = true
			if targetStepIdx == -1 {
				targetStepIdx = i
			}
		}

		if found {
			for _, id := range depts {
				if p, ok := state.Progress[id]; ok {
					if id == fromDeptID {
						p.CurrentStage = fromStage
					} else {
						// Reset others to their very first stage
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
	}

	if targetStepIdx != -1 {
		state.CurrentStep = targetStepIdx
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
