package workflow

import (
	"context"
	"fmt"
	"github.com/vncats/otel-demo/internal/store"
	"github.com/vncats/otel-demo/pkg/prim"
	"time"

	"go.temporal.io/sdk/workflow"
)

type TrackUserActionRequest struct {
}

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
	fmt.Println("compose action", payload)
	return &store.UserAction{}, nil
}

func (a *Activities) CreateAction(ctx context.Context, act *store.UserAction) error {
	fmt.Println("create action", act)
	return nil
}
