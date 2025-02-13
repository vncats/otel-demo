package tracing

import (
	"context"
	"fmt"
	"strconv"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// WrapConsumer wraps a kafka.Consumer so that any consumed events are traced.
func WrapConsumer(c *kafka.Consumer, opts WrapOptions) *Consumer {
	return &Consumer{
		Consumer:  c,
		spanAttrs: opts.SpanAttrs,
	}
}

type Consumer struct {
	*kafka.Consumer
	prev      trace.Span
	spanAttrs []attribute.KeyValue
}

func (c *Consumer) Poll(timeoutMs int) (event kafka.Event) {
	if c.prev != nil {
		c.prev.End()
		c.prev = nil
	}

	evt := c.Consumer.Poll(timeoutMs)
	if msg, ok := evt.(*kafka.Message); ok {
		c.prev = c.startSpan(msg)
	}

	return evt
}

func (c *Consumer) Close() error {
	err := c.Consumer.Close()
	if c.prev != nil {
		if c.prev.IsRecording() {
			c.prev.End()
		}
		c.prev = nil
	}
	return err
}

func (c *Consumer) startSpan(msg *kafka.Message) trace.Span {
	carrier := NewMessageCarrier(msg)
	parentCtx := propagator.Extract(context.Background(), carrier)

	attrs := []attribute.KeyValue{
		semconv.MessagingOperationTypeReceive,
		semconv.MessagingSystemKafka,
		semconv.MessagingKafkaMessageOffset(int(msg.TopicPartition.Offset)),
		semconv.MessagingKafkaMessageKey(string(msg.Key)),
		semconv.MessagingDestinationName(*msg.TopicPartition.Topic),
		semconv.MessagingMessageID(strconv.FormatInt(int64(msg.TopicPartition.Offset), 10)),
		semconv.MessagingDestinationPartitionID(strconv.Itoa(int(msg.TopicPartition.Partition))),
		semconv.MessagingMessageBodySize(getMsgSize(msg)),
	}
	attrs = append(attrs, c.spanAttrs...)

	spanCtx, span := tracer.Start(
		parentCtx,
		fmt.Sprintf("receive %s", *msg.TopicPartition.Topic),
		trace.WithNewRoot(),
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithLinks(trace.Link{SpanContext: trace.SpanContextFromContext(parentCtx)}),
	)
	propagator.Inject(spanCtx, carrier)

	return span
}
