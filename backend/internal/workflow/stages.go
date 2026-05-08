package workflow

import (
	"time"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/workflow"
)


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

		done, wasRejected, backToPrep, err := processStage(ctx, dept, stage, state, progress, transitionChan, commentChan)
		if err != nil {
			return false, err
		}

		if wasRejected {
			return true, nil
		}

		if backToPrep {
			stageIdx = findStageIndex(dept, StagePrep)
			progress.StageStatus = StageStatusPending
			progress.CurrentStage = StagePrep
		} else if done {
			stageIdx++
		}
	}

	return false, nil
}

func processStage(
	ctx workflow.Context,
	dept DepartmentDef,
	stage StageDef,
	state *WorkflowState,
	progress *DepartmentProgress,
	transitionChan, commentChan workflow.ReceiveChannel,
) (done bool, rejected bool, backToPrep bool, err error) {
	logger := workflow.GetLogger(ctx)
	for {
		selector := workflow.NewSelector(ctx)

		selector.AddReceive(transitionChan, func(c workflow.ReceiveChannel, _ bool) {
			var sig TransitionSignal
			c.Receive(ctx, &sig)

			if sig.DeptID != dept.ID {
				logger.Warn("Signal for different dept, skipping", "for", sig.DeptID, "current", dept.ID)
				return
			}

			switch sig.ToStage {
			case StageApprove:
				handleApproveTransition(logger, stage, progress, state, &done, &backToPrep)
			case StageReview:
				handleReviewTransition(logger, stage, progress, state, &done)
			case "reject":
				handleRejectTransition(dept, progress, state, &done, &rejected)
			}
		})

		selector.AddReceive(commentChan, func(c workflow.ReceiveChannel, _ bool) {
			handleCommentSignal(ctx, dept.ID, progress, commentChan)
		})

		selector.Select(ctx)
		if ctx.Err() != nil {
			err = ctx.Err()
			return
		}

		if done {
			return
		}
	}
}

func handleApproveTransition(logger log.Logger, stage StageDef, progress *DepartmentProgress, state *WorkflowState, done, backToPrep *bool) {

	switch stage.Type {
	case StageReview:
		if progress.HasComment {
			logger.Info("Comments found during review, routing back to prep", "dept", progress.DeptID)
			*backToPrep = true
			*done = true
		} else {
			progress.StageStatus = StageStatusDone
			state.UpdateWorkload()
			*done = true
		}
	case StageApprove:
		if stage.RequiresComment && !progress.HasComment {
			logger.Warn("Cannot approve without a comment", "dept", progress.DeptID)
			return
		}
		progress.StageStatus = StageStatusDone
		state.UpdateWorkload()
		*done = true
	default:
		logger.Warn("Cannot approve/advance to approve from this stage", "stage", stage.Type)
	}
}

func handleReviewTransition(logger log.Logger, stage StageDef, progress *DepartmentProgress, state *WorkflowState, done *bool) {

	if stage.Type != StagePrep {
		logger.Warn("Unexpected review transition", "stage", stage.Type)
		return
	}
	progress.StageStatus = StageStatusDone
	state.UpdateWorkload()
	*done = true
}

func handleRejectTransition(dept DepartmentDef, progress *DepartmentProgress, state *WorkflowState, done, rejected *bool) {
	progress.StageStatus = StageStatusRejected
	state.RejectedBy = dept.ID
	state.Status = WorkflowPaused
	state.UpdateWorkload()
	*done = true
	*rejected = true
}

func handleCommentSignal(ctx workflow.Context, deptID string, progress *DepartmentProgress, commentChan workflow.ReceiveChannel) {
	var sig CommentSignal
	commentChan.Receive(ctx, &sig)
	if sig.DeptID != deptID {
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
}
