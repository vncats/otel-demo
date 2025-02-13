package workflow

import (
	"context"
	"time"

	"github.com/vncats/otel-demo/internal/store"
	"github.com/vncats/otel-demo/pkg/prim"

	"go.temporal.io/sdk/workflow"
)

type TrackUserActionRequest struct{}

func TrackUserActionWorkflow(ctx workflow.Context, payload prim.Map) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})

	var acts *Activities

	act := &store.UserAction{}

	err := workflow.ExecuteActivity(ctx, acts.ComposeAction, payload).Get(ctx, act)
	if err != nil {
		return err
	}

	err = workflow.ExecuteActivity(ctx, acts.CreateAction, act).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}

type Activities struct {
	store store.IStore
}

func (a *Activities) ComposeAction(ctx context.Context, payload prim.Map) (*store.UserAction, error) {
	return &store.UserAction{Payload: prim.Map{
		"uid": payload.String("user_id"),
		"rid": payload.String("request_id"),
		"act": payload.String("action"),
		"ua":  payload.String("user_agent"),
		"ts":  time.Now().Unix(),
	}}, nil
}

func (a *Activities) CreateAction(ctx context.Context, act *store.UserAction) error {
	return a.store.CreateUserAction(ctx, act)
}
