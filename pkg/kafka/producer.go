package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/vncats/otel-demo/pkg/kafka/tracing"
	"go.opentelemetry.io/otel"
)

type ProducerClient interface {
	Events() chan kafka.Event
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error
	Flush(timeoutMs int) int
	Close()
}

type ProducerOptions struct {
	Brokers         string
	CompressionType string
	BatchSize       int
	BatchMessages   int
	LingerMs        int

	EnableTracing bool
}

func (o ProducerOptions) KafkaConfig() kafka.ConfigMap {
	cm := kafka.ConfigMap{
		"bootstrap.servers":  o.Brokers,
		"batch.num.messages": 50,
		"batch.size":         16384,
		"linger.ms":          10,
		"compression.type":   "snappy",
	}
	setKafkaConfig(cm, "batch.size", o.BatchSize)
	setKafkaConfig(cm, "batch.num.messages", o.BatchMessages)
	setKafkaConfig(cm, "linger.ms", o.LingerMs)
	setKafkaConfig(cm, "compression.type", o.CompressionType)

	return cm
}

func setKafkaConfig[T comparable](m kafka.ConfigMap, key string, v T) {
	var zeroValue T
	if v == zeroValue {
		return
	}
	m[key] = v
}

type MessageOption func(*kafka.Message) *kafka.Message

func WithTraceContext(ctx context.Context) MessageOption {
	return func(msg *kafka.Message) *kafka.Message {
		carrier := tracing.NewMessageCarrier(msg)
		otel.GetTextMapPropagator().Inject(ctx, carrier)
		return msg
	}
}

type Producer struct {
	client ProducerClient
}

func NewProducer(opts ProducerOptions) (*Producer, error) {
	kkConfig := opts.KafkaConfig()

	c, err := kafka.NewProducer(&kkConfig)
	if err != nil {
		return nil, err
	}

	var client ProducerClient = c
	if opts.EnableTracing {
		client = tracing.WrapProducer(c, tracing.WrapOptions{
			SpanAttrs: tracing.GetKafkaAttrs(kkConfig),
		})
	}

	producer := &Producer{
		client: client,
	}

	return producer, nil
}

func (p *Producer) Produce(topic string, key string, value any, opts ...MessageOption) (*kafka.Message, error) {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            []byte(key),
		Value:          valueBytes,
	}
	for _, opt := range opts {
		msg = opt(msg)
	}

	deliveryChan := make(chan kafka.Event)
	if err = p.client.Produce(msg, deliveryChan); err != nil {
		return nil, err
	}

	ev := <-deliveryChan
	if delivered, ok := ev.(*kafka.Message); ok {
		if delivered.TopicPartition.Error != nil {
			return nil, msg.TopicPartition.Error
		}

		return delivered, nil
	}

	return nil, fmt.Errorf("unexpected event type: %T", ev)
}

func (p *Producer) Start() {
	// Handle events on default delivery channel
	go func() {
		for e := range p.client.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					fmt.Printf("Delivery failed: %v\n", ev.TopicPartition)
				} else {
					fmt.Printf("Delivered message to %v\n", ev.TopicPartition)
				}
			}
		}
	}()
}

func (p *Producer) Stop() {
	p.client.Flush(5000)
	p.client.Close()
}
