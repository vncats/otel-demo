package workflow

import (
	"github.com/vncats/otel-demo/internal/store"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

const (
	TaskQueue = "my_service_task_queue"
)

func NewWorker(c client.Client, store store.IStore) worker.Worker {
	w := worker.New(c, TaskQueue, worker.Options{})
	w.RegisterWorkflow(TrackUserActionWorkflow)

	acts := &Activities{
		store: store,
	}
	w.RegisterActivity(acts.ComposeAction)
	w.RegisterActivity(acts.CreateAction)

	return w
}
