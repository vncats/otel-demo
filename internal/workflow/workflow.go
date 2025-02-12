package workflow

import (
	"context"
	"fmt"
	"github.com/vncats/otel-demo/internal/store"
	"github.com/vncats/otel-demo/pkg/prim"
	"time"

	"go.temporal.io/sdk/workflow"
)

type LogActivityRequest struct {
}

func LogActivityWorkflow(ctx workflow.Context, payload prim.Map) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})

	var acts *Activities

	act := &store.UserActivity{}

	err := workflow.ExecuteActivity(ctx, acts.ComposeActivity, payload).Get(ctx, act)
	if err != nil {
		return err
	}

	err = workflow.ExecuteActivity(ctx, acts.CreateActivity, act).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}

type Activities struct {
	store store.IStore
}

func (a *Activities) ComposeActivity(ctx context.Context, payload prim.Map) (*store.UserActivity, error) {
	fmt.Println("compose activity")
	return &store.UserActivity{}, nil
}

func (a *Activities) CreateActivity(ctx context.Context, act *store.UserActivity) error {
	fmt.Println("create activity")
	return nil
}
