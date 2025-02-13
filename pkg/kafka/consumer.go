package kafka

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/vncats/otel-demo/pkg/kafka/tracing"
	"github.com/vncats/otel-demo/pkg/otel/log"
	"github.com/vncats/otel-demo/pkg/retry"
	"sync"
)

const (
	OffsetEarliest = "earliest"
	OffsetLatest   = "latest"
)

type ConsumerClient interface {
	Poll(timeoutMs int) kafka.Event
	Close() error
}

type ConsumerOptions struct {
	Brokers string
	Group   string
	Topics  []string
	Offset  string

	EnableTracing bool

	MessageHandler func(msg *kafka.Message) error
	ErrorHandler   func(err kafka.Error) error
	OtherHandler   func(ev kafka.Event) error
}

type Consumer struct {
	client ConsumerClient

	messageHandler func(msg *kafka.Message) error
	errorHandler   func(err kafka.Error) error
	otherHandler   func(ev kafka.Event) error

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

func NewConsumer(opts ConsumerOptions) (*Consumer, error) {
	if opts.MessageHandler == nil {
		opts.MessageHandler = noopMessageHandler
	}

	if opts.ErrorHandler == nil {
		opts.ErrorHandler = noopErrorHandler
	}

	if opts.OtherHandler == nil {
		opts.OtherHandler = noopOtherHandler
	}

	kkConfig := kafka.ConfigMap{
		"bootstrap.servers": opts.Brokers,
		"group.id":          opts.Group,
		"auto.offset.reset": opts.Offset,
	}
	c, err := kafka.NewConsumer(&kkConfig)
	if err != nil {
		return nil, err
	}

	if err = c.SubscribeTopics(opts.Topics, nil); err != nil {
		return nil, err
	}

	var client ConsumerClient = c
	if opts.EnableTracing {
		client = tracing.WrapConsumer(c, tracing.WrapOptions{
			SpanAttrs: tracing.GetKafkaAttrs(kkConfig),
		})
	}

	consumer := &Consumer{
		client:         client,
		messageHandler: opts.MessageHandler,
		errorHandler:   opts.ErrorHandler,
		otherHandler:   opts.OtherHandler,
	}

	return consumer, nil
}

func (c *Consumer) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.wg.Add(1)

	log.Info(ctx, "Starting consumer")
	go func() {
		defer c.wg.Done()
		for {
			ev := c.client.Poll(1000)
			if ev == nil {
				continue
			}

			c.handleEvent(ev)

			// stop if ctx cancel
			if ctx.Err() != nil {
				break
			}
		}
	}()
}

func (c *Consumer) Stop() {
	ctx := context.Background()

	log.Info(ctx, "Stopping consumer")
	c.cancel()

	log.Info(ctx, "Waiting consumer goroutines to complete")
	c.wg.Wait()

	log.Info(ctx, "Closing consumer")
	if err := c.client.Close(); err != nil {
		log.Error(ctx, "failed to close consumer", err)
	}

	log.Info(ctx, "Stopped consumer")
}

func (c *Consumer) handleEvent(ev kafka.Event) {
	defer func() {
		if r := recover(); r != nil {
			log.Error(context.Background(), fmt.Sprintf("panic: %v", r))
		}
	}()

	// handle message
	switch e := ev.(type) {
	case *kafka.Message:
		_ = c.messageHandler(e)
	case kafka.Error:
		_ = c.errorHandler(e)
	default:
		_ = c.otherHandler(e)
	}
}

func HandleWithRetry[T kafka.Event](fn func(T) error, cfg retry.Config) func(T) error {
	return func(t T) error {
		return retry.Do(func() error {
			return fn(t)
		}, cfg)
	}
}

func noopMessageHandler(_ *kafka.Message) error {
	return nil
}

func noopErrorHandler(_ kafka.Error) error {
	return nil
}

func noopOtherHandler(_ kafka.Event) error {
	return nil
}
