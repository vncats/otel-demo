package kafka

import (
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/vncats/otel-demo/pkg/kafka/tracing"
	"go.opentelemetry.io/otel/baggage"
)

type ProducerClient interface {
	Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error
	Close()
}

type ProducerOptions struct {
	Brokers         string
	BatchSize       int
	LingerMs        int
	CompressionType string

	EnableTracing bool
	TraceBaggage  *baggage.Baggage
}

func (o ProducerOptions) KafkaConfig() kafka.ConfigMap {
	cm := kafka.ConfigMap{
		"bootstrap.servers": o.Brokers,
		"batch.size":        20,
		"linger.ms":         1000,
		"compression.type":  "snappy",
	}
	setKafkaConfig(cm, "batch.size", o.BatchSize)
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

func (c *Producer) Produce(topic string, key string, value any) (*kafka.Message, error) {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	kafkaMessage := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Key:            []byte(key),
		Value:          valueBytes,
	}

	deliveryChan := make(chan kafka.Event)
	if err = c.client.Produce(kafkaMessage, deliveryChan); err != nil {
		return nil, err
	}

	r := <-deliveryChan
	m, ok := r.(*kafka.Message)
	if !ok {
		return nil, fmt.Errorf("type assertion failed: %T", r)
	}

	if m.TopicPartition.Error != nil {
		return nil, m.TopicPartition.Error
	}

	return m, nil
}

func (c *Producer) Close() {
	c.client.Close()
}
