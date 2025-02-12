package kafka

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/vncats/otel-demo/pkg/otel/log"
	"sync"
)

type ConsumerConfig struct {
	Brokers string   `yaml:"brokers"`
	Group   string   `yaml:"group"`
	Topics  []string `yaml:"topics"`
	Offset  string   `yaml:"offset"`
}

type Consumer struct {
	consumer *kafka.Consumer
	config   ConsumerConfig

	messageHandler func(msg *kafka.Message)
	errorHandler   func(err kafka.Error)
	otherHandler   func(ev kafka.Event)

	wg     sync.WaitGroup
	cancel context.CancelFunc
}

type Option func(c *Consumer)

func WithMessageHandler(fn func(msg *kafka.Message)) Option {
	return func(c *Consumer) {
		c.messageHandler = fn
	}
}

func WithErrorHandler(fn func(err kafka.Error)) Option {
	return func(c *Consumer) {
		c.errorHandler = fn
	}
}

func WithOtherHandler(fn func(ev kafka.Event)) Option {
	return func(c *Consumer) {
		c.otherHandler = fn
	}
}

func NewConsumer(cfg ConsumerConfig, opts ...Option) (*Consumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": cfg.Brokers,
		"group.id":          cfg.Group,
		"auto.offset.reset": cfg.Offset,
	})
	if err != nil {
		return nil, err
	}

	if err = c.SubscribeTopics(cfg.Topics, nil); err != nil {
		return nil, err
	}

	consumer := &Consumer{
		config:         cfg,
		consumer:       c,
		messageHandler: noopMessageHandler,
		errorHandler:   noopErrorHandler,
		otherHandler:   noopOtherHandler,
	}
	for _, opt := range opts {
		opt(consumer)
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
			ev := c.consumer.Poll(1000)
			if ev == nil {
				return
			}

			c.handleEvent(ev)

			// stop if ctx cancel
			if ctx.Err() != nil {
				return
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
	if err := c.consumer.Close(); err != nil {
		log.Error(ctx, "failed to close consumer group", err)
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
		c.messageHandler(e)
	case kafka.Error:
		c.errorHandler(e)
		if e.Code() == kafka.ErrAllBrokersDown {
			break
		}
	default:
		c.otherHandler(e)
	}
}

//func (c *Consumer) handle(ctx context.Context, msg *kafka.Message) error {
//	ctx = log.WithContext(ctx, "topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset, "timestamp", msg.Timestamp)
//	backoff := NewBackoff(ctx, BackoffConfig{
//		MinInterval:   1 * time.Second,
//		MaxInterval:   10 * time.Second,
//		BackoffFactor: 2.0,
//		MaxRetries:    5,
//	})
//	for {
//		err := h.handleFn(ctx, msg)
//		if err == nil {
//			return nil
//		}
//		log.Error(ctx, "failed to handle message", err)
//
//		backoff.Backoff()
//		if backoff.GiveUp() {
//			break
//		}
//	}
//
//	log.Info(ctx, "give up handling message")
//
//	return nil
//}

func noopMessageHandler(_ *kafka.Message) {
	return
}

func noopErrorHandler(_ kafka.Error) {
	return
}

func noopOtherHandler(_ kafka.Event) {
	return
}
