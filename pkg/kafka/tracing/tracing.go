package tracing

import (
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const scopeName = "github.com/confluentinc/confluent-kafka-go"

var (
	propagator = otel.GetTextMapPropagator()
	tracer     = otel.Tracer(scopeName)
	meter      = otel.Meter(scopeName)
)

type WrapOptions struct {
	Attributes []attribute.KeyValue
}

func GetKafkaAttrs(cm kafka.ConfigMap) []attribute.KeyValue {
	var attrs []attribute.KeyValue
	if group, err := cm.Get("group.id", ""); err == nil {
		attrs = append(attrs, semconv.MessagingKafkaConsumerGroup(group.(string)))
	}
	if bs, err := cm.Get("bootstrap.servers", ""); err == nil && bs != "" {
		attrs = append(attrs, semconv.ServerAddress(bs.(string)))
	}
	return attrs
}

// getMsgSize returns the size of a kafka message
func getMsgSize(msg *kafka.Message) (size int) {
	for _, header := range msg.Headers {
		size += len(header.Key) + len(header.Value)
	}
	return size + len(msg.Value) + len(msg.Key)
}
