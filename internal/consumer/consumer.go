package consumer

import (
	"context"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"runtime/trace"
)

type Options struct {
	Group   string
	Topic   string
	Brokers string
	Handler func(ctx context.Context, msg *kafka.Message)
}

type Consumer struct {
	consumer *kafka.Consumer
}

// NewConsumer returns new instance
func NewConsumer(opts Options) (*Consumer, error) {
	handler := func(msg *ckafka.Message) {
		ctx, span := trace.Start(msg, "process")
		defer span.End()

		opts.Handler(ctx, msg)
	}

	builder := kafka.NewConsumerBuilder().
		WithConsumerGroup(opts.Group).
		WithOffset(kafka.OffsetEarliest).
		WithPollingTimeOut(5000).
		WithEventCallbacks(handler, kafka.DefaultErrorCallback).
		WithoutLogNoEvent().
		WithTracing()

	builder.WithBootstrapServers(opts.Brokers)

	consumer, err := builder.Build()
	if err != nil {
		return nil, err
	}

	err = consumer.Subscribe([]string{opts.Topic})
	if err != nil {
		return nil, err
	}

	return &Consumer{consumer: consumer}, nil
}

// Start starts all consumers
func (s *Consumer) Start() {
	s.consumer.Start()
}

// Stop stops  all consumers
func (s *Consumer) Stop() {
	_ = s.consumer.Close()
}
