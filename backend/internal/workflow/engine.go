package workflow

import (
	"fmt"
	"go.temporal.io/sdk/workflow"
)

func DSLWorkflow(ctx workflow.Context, def WorkflowDef) (string, error) {
	logger := workflow.GetLogger(ctx)

	state := initState(ctx, def)

	if err := workflow.SetQueryHandler(ctx, QueryStatus, func() (*WorkflowState, error) {
		return state, nil
	}); err != nil {
		return "", fmt.Errorf("failed to set query handler: %w", err)
	}

	deptTransitionChans, deptCommentChans := setupSignalHandlers(ctx, def)

	adminStartChan := workflow.GetSignalChannel(ctx, AdminStartChannel)
	adminChan := workflow.GetSignalChannel(ctx, AdminRoutingChannel)

	logger.Info("Workflow pending admin assignment", "name", def.Name)
	var startSig AdminStartSignal
	adminStartChan.Receive(ctx, &startSig)
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	logger.Info("Admin assigned identities and started workflow", "admin", startSig.AdminID)
	state.ApplyAssignments(startSig.Assignments)
	state.Status = WorkflowRunning
	state.UpdateWorkload()

	if err := executePlan(ctx, def, state, deptTransitionChans, deptCommentChans, adminChan); err != nil {
		return "", err
	}

	if state.Status == WorkflowRejected {
		return "Workflow rejected", nil
	}

	state.Status = WorkflowApproved
	logger.Info("Workflow fully approved", "name", def.Name)
	return "Workflow Fully Approved", nil
}

func executePlan(ctx workflow.Context, def WorkflowDef, state *WorkflowState, transitionChans, commentChans map[string]workflow.Channel, adminChan workflow.ReceiveChannel) error {
OuterLoop:
	for {
		for stepIdx := state.CurrentStep; stepIdx < len(def.Execution.Steps); stepIdx++ {
			step := def.Execution.Steps[stepIdx]
			state.CurrentStep = stepIdx

			rejected, err := executeStep(ctx, step, def, state, transitionChans, commentChans, adminChan)
			if err != nil {
				return err
			}
			if rejected {
				if state.Status == WorkflowRejected {
					return nil
				}
				continue OuterLoop
			}
		}
		break
	}
	return nil
}

func executeStep(ctx workflow.Context, step ExecutionStep, def WorkflowDef, state *WorkflowState, transitionChans, commentChans map[string]workflow.Channel, adminChan workflow.ReceiveChannel) (bool, error) {
	if len(step.Sequential) > 0 {
		return handleSequentialStep(ctx, step, def, state, transitionChans, commentChans, adminChan)
	}
	if len(step.Parallel) > 0 {
		return handleParallelStep(ctx, step, def, state, transitionChans, commentChans, adminChan)
	}
	if len(step.Exclusive) > 0 {
		return handleExclusiveStep(ctx, step, def, state, transitionChans, commentChans, adminChan)
	}
	return false, nil
}

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

