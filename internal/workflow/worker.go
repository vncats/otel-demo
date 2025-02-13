package workflow

import (
	"log/slog"
	"os"

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

func NewWorker(c client.Client) worker.Worker {
	w := worker.New(c, TaskQueue, worker.Options{})
	w.RegisterWorkflow(TrackUserActionWorkflow)

	acts := &Activities{}
	w.RegisterActivity(acts.ComposeAction)
	w.RegisterActivity(acts.CreateAction)

	return w
}
