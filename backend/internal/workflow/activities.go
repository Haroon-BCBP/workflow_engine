package workflow

import (
	"context"

	"go.temporal.io/sdk/activity"
)

type Activities struct{}

func (a *Activities) StageStartedActivity(ctx context.Context, deptID, stage string) error {
	log := activity.GetLogger(ctx)
	log.Info("Stage started", "dept", deptID, "stage", stage)
	return nil
}

func (a *Activities) SaveCommentActivity(ctx context.Context, deptID, stage, userID, text string) error {
	log := activity.GetLogger(ctx)
	log.Info("Comment saved", "dept", deptID, "stage", stage, "user", userID)
	return nil
}

func StageStartedActivity(ctx context.Context, deptID, stage string) error {
	return (&Activities{}).StageStartedActivity(ctx, deptID, stage)
}

func SaveCommentActivity(ctx context.Context, deptID, stage, userID, text string) error {
	return (&Activities{}).SaveCommentActivity(ctx, deptID, stage, userID, text)
}
