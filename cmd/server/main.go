package main

import (
	"context"
	"github.com/vncats/otel-demo/internal/server"

	"github.com/vncats/otel-demo/internal/cache"
	"github.com/vncats/otel-demo/internal/message"
	"github.com/vncats/otel-demo/internal/store"
	"github.com/vncats/otel-demo/internal/workflow"
	"github.com/vncats/otel-demo/pkg/otel/log"
	"github.com/vncats/otel-demo/pkg/otel/metric"
	"github.com/vncats/otel-demo/pkg/otel/sdk"
)

func main() {
	ctx := context.Background()

	// Set up OTel SDK
	shutdownFn, err := sdk.SetupFromYAML(ctx)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = shutdownFn(ctx)
	}()

	// Setup log
	log.Setup("movie_service")

	// Start runtime metrics.
	err = metric.StartRuntime()
	if err != nil {
		panic(err)
	}

	// Migrate database
	st, err := store.NewStore("admin:password@tcp(127.0.0.1:3306)/movie_db?charset=utf8mb4&parseTime=True&loc=Local")
	if err != nil {
		panic(err)
	}
	if err = st.Migrate(); err != nil {
		panic(err)
	}

	// New cache
	cs, err := cache.NewCache("redis://:password@localhost:6379/1", st)
	if err != nil {
		panic(err)
	}
	defer cs.Close()

	// New consumer
	consumer, err := message.NewStatsConsumer(st)
	if err != nil {
		panic(err)
	}
	consumer.Start()
	defer consumer.Stop()

	// New producer
	producer, err := message.NewProducer("localhost:9092")
	if err != nil {
		panic(err)
	}
	producer.Start()
	defer producer.Stop()

	tc, err := workflow.NewClient()
	if err != nil {
		panic(err)
	}
	defer tc.Close()

	// New worker
	w := workflow.NewWorker(tc, st)
	if err = w.Start(); err != nil {
		panic(err)
	}
	defer w.Stop()

	h := server.NewHandler(st, producer, cs, tc)
	s := server.NewServer(h)
	s.Start()
}
