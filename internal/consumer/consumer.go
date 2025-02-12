package consumer

//
//import (
//	"context"
//	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
//	"github.com/vncats/otel-demo/pkg/otel/log"
//	"sync"
//	"time"
//)
//
//type Config struct {
//	Brokers       string   `yaml:"brokers"`
//	Group         string   `yaml:"group"`
//	Topics        []string `yaml:"topics"`
//	InitialOffset string   `yaml:"initial_offset"`
//}
//
//type Consumer struct {
//	consumer *kafka.Consumer
//	config   Config
//
//	wg     sync.WaitGroup
//	cancel context.CancelFunc
//}
//
//type HandlerFunc func(ctx context.Context, msg *kafka.Message) error
//
//func NewConsumer(cfg Config) (*Consumer, error) {
//	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
//		"bootstrap.servers": cfg.Brokers,
//		"group.id":          cfg.Group,
//		"auto.offset.reset": cfg.InitialOffset,
//	})
//	if err != nil {
//		return nil, err
//	}
//
//	err = consumer.SubscribeTopics(cfg.Topics, nil)
//	//consumer := &Consumer{
//	//	handleFn: handleFn,
//	//	ready:    make(chan bool),
//	//}
//
//	return &Consumer{
//		config:   cfg,
//		consumer: consumer,
//	}, nil
//}
//
//func (c *Consumer) Start() {
//	ctx, cancel := context.WithCancel(context.Background())
//	c.cancel = cancel
//	c.wg.Add(1)
//
//	log.Info(ctx, "Starting consumer")
//	go func() {
//		defer c.wg.Done()
//		for {
//			if err := c.group.Consume(ctx, c.config.Topics, c.consumer); err != nil {
//				log.Error(ctx, "failed to consume message", err)
//			}
//			if ctx.Err() != nil {
//				return
//			}
//
//			c.consumer.ready = make(chan bool)
//		}
//	}()
//
//	<-c.consumer.ready
//	log.Info(ctx, "Consumer is up and running...")
//}
//
//func (c *Consumer) Stop() {
//	ctx := context.Background()
//
//	log.Info(ctx, "Stopping consumer")
//	c.cancel()
//
//	log.Info(ctx, "Waiting consumer goroutines to complete")
//	c.wg.Wait()
//
//	log.Info(ctx, "Stopping consumer handler and group")
//	if err := c.group.Close(); err != nil {
//		log.Error(ctx, "failed to close consumer group", err)
//	}
//
//	log.Info(ctx, "Stopped consumer")
//}
//
//func (h *Consumer) handle(ctx context.Context, msg *kafka.Message) error {
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
