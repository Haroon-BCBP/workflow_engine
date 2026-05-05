package dsl

import (
	"context"

	"go.temporal.io/sdk/activity"
)

// Activities holds all activity implementations for the DSL workflow.
// All are generic — dept_id and stage differentiate them at runtime.
type Activities struct{}

// StageStartedActivity is a notification/logging hook called when a stage begins.
func (a *Activities) StageStartedActivity(ctx context.Context, deptID, stage string) error {
	log := activity.GetLogger(ctx)
	log.Info("Stage started", "dept", deptID, "stage", stage)
	// TODO: integrate with notification service
	return nil
}

func (a *Activities) SaveCommentActivity(ctx context.Context, deptID, stage, userID, text string) error {
	log := activity.GetLogger(ctx)
	log.Info("Comment saved", "dept", deptID, "stage", stage, "user", userID)
	// TODO: persist to DB via repository
	return nil
}

func StageStartedActivity(ctx context.Context, deptID, stage string) error {
	return (&Activities{}).StageStartedActivity(ctx, deptID, stage)
}

func SaveCommentActivity(ctx context.Context, deptID, stage, userID, text string) error {
	return (&Activities{}).SaveCommentActivity(ctx, deptID, stage, userID, text)
}
