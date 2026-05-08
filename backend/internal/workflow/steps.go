package workflow

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func handleSequentialStep(ctx workflow.Context, step ExecutionStep, def WorkflowDef, state *WorkflowState, transitionChans, commentChans map[string]workflow.Channel, adminChan workflow.ReceiveChannel) (bool, error) {
	logger := workflow.GetLogger(ctx)
	for _, deptID := range step.Sequential {
		dept := findDept(def, deptID)
		if dept == nil {
			return false, fmt.Errorf("dept %q not found in definition", deptID)
		}
		rejected, err := processDepartment(ctx, *dept, def, state, transitionChans[deptID], commentChans[deptID])
		if err != nil {
			return false, err
		}
		if rejected {
			logger.Info("Workflow paused for admin routing", "dept", deptID)
			routed, err := waitForAdminRouting(ctx, def, state, adminChan)
			if err != nil || !routed {
				state.Status = WorkflowRejected
				return true, nil
			}
			return true, nil
		}
	}
	return false, nil
}

func handleParallelStep(ctx workflow.Context, step ExecutionStep, def WorkflowDef, state *WorkflowState, transitionChans, commentChans map[string]workflow.Channel, adminChan workflow.ReceiveChannel) (bool, error) {
	logger := workflow.GetLogger(ctx)
	rejected, err := runParallel(ctx, step.Parallel, def, state, transitionChans, commentChans)
	if err != nil {
		return false, err
	}
	if rejected {
		logger.Info("Workflow paused for admin routing (parallel rejection)")
		routed, err := waitForAdminRouting(ctx, def, state, adminChan)
		if err != nil || !routed {
			state.Status = WorkflowRejected
			return true, nil
		}
		return true, nil
	}
	return false, nil
}

func handleExclusiveStep(ctx workflow.Context, step ExecutionStep, def WorkflowDef, state *WorkflowState, transitionChans, commentChans map[string]workflow.Channel, adminChan workflow.ReceiveChannel) (bool, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Workflow paused for XOR routing", "options", step.Exclusive)
	state.Status = WorkflowPausedXOR

	var chosenDept string
	for {
		var sig AdminRoutingSignal
		adminChan.Receive(ctx, &sig)
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		if sig.Action == "xor_route" {
			chosenDept = sig.DeptID
			break
		} else if sig.Action == "terminate" {
			state.Status = WorkflowRejected
			return true, nil
		}
	}

	state.Status = WorkflowRunning
	logger.Info("Admin chose XOR branch", "dept", chosenDept)

	dept := findDept(def, chosenDept)
	if dept == nil {
		return false, fmt.Errorf("dept %q not found in definition", chosenDept)
	}
	rejected, err := processDepartment(ctx, *dept, def, state, transitionChans[chosenDept], commentChans[chosenDept])
	if err != nil {
		return false, err
	}
	if rejected {
		logger.Info("Workflow paused for admin routing (xor rejection)", "dept", chosenDept)
		routed, err := waitForAdminRouting(ctx, def, state, adminChan)
		if err != nil || !routed {
			state.Status = WorkflowRejected
			return true, nil
		}
		return true, nil
	}
	return false, nil
}

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

		future, settable := workflow.NewFuture(childCtx)
		workflow.Go(childCtx, func(ctx workflow.Context) {
			rejected, err := processDepartment(ctx, *dept, def, state, deptTransitionChans[deptID], deptCommentChans[deptID])
			settable.Set(rejected, err)
		})
		selector.AddFuture(future, func(f workflow.Future) {
			completedCount++
			var rejected bool
			if err := f.Get(ctx, &rejected); err != nil {
				if firstErr == nil && !temporal.IsCanceledError(err) {
					firstErr = err
				}
				return
			}
			if rejected {
				wasRejected = true
			}
		})
	}

	for completedCount < len(deptIDs) {
		selector.Select(ctx)
		if firstErr != nil {
			return false, firstErr
		}
		if wasRejected {
			drainSiblingBranches(ctx, selector, cancel, &completedCount, len(deptIDs))
			return true, nil
		}
	}

	return false, nil
}

func drainSiblingBranches(ctx workflow.Context, selector workflow.Selector, cancel workflow.CancelFunc, completedCount *int, total int) {
	cancel()
	timerExpired := false
	timerFuture := workflow.NewTimer(ctx, 2*time.Second)
	selector.AddFuture(timerFuture, func(f workflow.Future) {
		timerExpired = true
	})
	for *completedCount < total && !timerExpired {
		selector.Select(ctx)
	}
}
