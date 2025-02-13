package message

import (
	"github.com/vncats/otel-demo/pkg/kafka"
)

func NewProducer(brokers string) (*kafka.Producer, error) {
	producer, err := kafka.NewProducer(kafka.ProducerOptions{
		Brokers:       brokers,
		EnableTracing: true,
	})
	if err != nil {
		return nil, err
	}

	return producer, nil
}
