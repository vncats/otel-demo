package producer

import "github.com/confluentinc/confluent-kafka-go/v2/kafka"

type IProducer interface {
	Produce(topic, key string, value []byte) error
}

var _ IProducer = (*Producer)(nil)

func NewProducer(brokers string) (*Producer, error) {
	producer, err := kafka.NewProducerBuilder().
		WithThroughputConfig(10, 50).
		WithBootstrapServers(brokers).
		WithTracing().
		Build()

	if err != nil {
		return nil, err
	}

	return &Producer{producer}, nil
}

type Producer struct {
	producer *kafka.Producer
}

func (p *Producer) Produce(topic, key string, value []byte) error {
	return p.producer.ProduceAsync(topic, []byte(key), value, nil)
}
