package workflow

import (
	"log/slog"
	"os"

	"github.com/vncats/otel-demo/internal/store"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	tlog "go.temporal.io/sdk/log"
	"go.temporal.io/sdk/worker"
)

const (
	TaskQueue = "my_service_task_queue"
)

func NewClient() (client.Client, error) {
	tracingInterceptor, err := opentelemetry.NewTracingInterceptor(opentelemetry.TracerOptions{})
	if err != nil {
		return nil, err
	}

	logger := tlog.NewStructuredLogger(
		slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		),
	)

	return client.Dial(client.Options{
		HostPort:     "localhost:7233",
		Logger:       logger,
		Interceptors: []interceptor.ClientInterceptor{tracingInterceptor},
	})
}

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
