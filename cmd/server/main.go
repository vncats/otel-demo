package main

import (
	"context"
	"github.com/vncats/otel-demo/internal/cache"
	"github.com/vncats/otel-demo/internal/consumer"
	"github.com/vncats/otel-demo/internal/handler"
	"github.com/vncats/otel-demo/internal/producer"
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
	c, err := cache.NewCache("redis://:password@localhost:6379/1", st)
	if err != nil {
		panic(err)
	}
	defer c.Close()

	// Start consumer
	cs, err := consumer.NewStatsConsumer(st)
	if err != nil {
		panic(err)
	}
	cs.Start()
	defer cs.Stop()

	p, err := producer.NewProducer("localhost:9092")
	if err != nil {
		panic(err)
	}

	tc, err := workflow.NewClient()
	if err != nil {
		panic(err)
	}
	defer tc.Close()

	w := workflow.NewWorker(tc)
	if err := w.Start(); err != nil {
		panic(err)
	}
	defer w.Stop()

	h := handler.NewHandler(st, p, c)

	s := handler.NewServer(h, tc)
	_ = s.Start(ctx)
}
