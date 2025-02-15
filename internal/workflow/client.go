package workflow

import (
	"log/slog"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/contrib/opentelemetry"
	"go.temporal.io/sdk/interceptor"
	tlog "go.temporal.io/sdk/log"
)

// NewClient create a new temporal client
func NewClient() (client.Client, error) {
	tracingInterceptor, err := opentelemetry.NewTracingInterceptor(opentelemetry.TracerOptions{})
	if err != nil {
		return nil, err
	}

	metricHandler := opentelemetry.NewMetricsHandler(opentelemetry.MetricsHandlerOptions{})

	logger := tlog.NewStructuredLogger(
		slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		),
	)

	return client.Dial(client.Options{
		HostPort:       "localhost:7233",
		MetricsHandler: metricHandler,
		Logger:         logger,
		Interceptors:   []interceptor.ClientInterceptor{tracingInterceptor},
	})
}
