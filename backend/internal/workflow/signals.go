package workflow

import (
	"fmt"
	"go.temporal.io/sdk/workflow"
)

func setupSignalHandlers(ctx workflow.Context, def WorkflowDef) (map[string]workflow.Channel, map[string]workflow.Channel) {
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
				relaySignal(ctx, c, deptTransitionChans, func(s TransitionSignal) string { return s.DeptID })
			})
			selector.AddReceive(commentChan, func(c workflow.ReceiveChannel, _ bool) {
				relaySignal(ctx, c, deptCommentChans, func(s CommentSignal) string { return s.DeptID })
			})
			selector.Select(ctx)
			if ctx.Err() != nil {
				return
			}
		}
	})

	return deptTransitionChans, deptCommentChans
}

func relaySignal[T any](ctx workflow.Context, c workflow.ReceiveChannel, chans map[string]workflow.Channel, getDeptID func(T) string) {
	var sig T
	c.Receive(ctx, &sig)
	deptID := getDeptID(sig)
	if ch, ok := chans[deptID]; ok {
		workflow.Go(ctx, func(ctx workflow.Context) {
			ch.Send(ctx, sig)
		})
	} else {
		workflow.GetLogger(ctx).Warn("Received signal for unknown dept", "dept", deptID, "type", fmt.Sprintf("%T", sig))
	}
}
