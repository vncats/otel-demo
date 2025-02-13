package message

import (
	"context"
	ckafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/vncats/otel-demo/pkg/kafka"
)

type IProducer interface {
	Produce(ctx context.Context, topic string, key string, value any) (*ckafka.Message, error)
	Start()
	Stop()
}

var _ IProducer = (*Producer)(nil)

func NewProducer(brokers string) (IProducer, error) {
	producer, err := kafka.NewProducer(kafka.ProducerOptions{
		Brokers:       brokers,
		EnableTracing: true,
	})
	if err != nil {
		return nil, err
	}

	return &Producer{producer: producer}, nil
}

type Producer struct {
	producer *kafka.Producer
}

func (p *Producer) Produce(ctx context.Context, topic string, key string, value any) (*ckafka.Message, error) {
	return p.producer.Produce(topic, key, value, kafka.WithTraceContext(ctx))
}

func (p *Producer) Start() {
	p.producer.Start()
}

func (p *Producer) Stop() {
	p.producer.Stop()
}
