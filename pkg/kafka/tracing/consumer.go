package tracing

import (
	"context"
	"fmt"
	"go.opentelemetry.io/otel/metric"
	"strconv"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// WrapConsumer wraps a kafka.Consumer so that any consumed events are traced.
func WrapConsumer(c *kafka.Consumer, opts WrapOptions) (*Consumer, error) {
	consumer := &Consumer{
		Consumer:  c,
		spanAttrs: opts.SpanAttrs,
	}
	if err := consumer.createMetrics(); err != nil {
		return nil, err
	}

	return consumer, nil
}

type Consumer struct {
	*kafka.Consumer

	spanAttrs       []attribute.KeyValue
	durationMeasure metric.Float64Histogram

	endPrevSpanFn func()
	lock          sync.Mutex
}

func (c *Consumer) Poll(timeoutMs int) (event kafka.Event) {
	c.endPrevSpan()

	evt := c.Consumer.Poll(timeoutMs)
	if msg, ok := evt.(*kafka.Message); ok {
		c.startSpan(msg)
	}

	return evt
}

func (c *Consumer) Close() error {
	err := c.Consumer.Close()
	c.endPrevSpan()

	return err
}

func (c *Consumer) startSpan(msg *kafka.Message) {
	c.lock.Lock()
	defer c.lock.Unlock()

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

	startTime := time.Now()
	operationName := fmt.Sprintf("receive %s", *msg.TopicPartition.Topic)

	spanCtx, span := tracer.Start(
		parentCtx,
		operationName,
		trace.WithNewRoot(),
		trace.WithTimestamp(startTime),
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithLinks(trace.Link{SpanContext: trace.SpanContextFromContext(parentCtx)}),
	)
	propagator.Inject(spanCtx, carrier)

	c.endPrevSpanFn = func() {
		span.End()
		opts := metric.WithAttributes(
			attribute.String("operation.name", operationName),
		)
		elapsedTime := float64(time.Since(startTime)) / float64(time.Millisecond)
		c.durationMeasure.Record(spanCtx, elapsedTime, opts)
	}
}

func (c *Consumer) endPrevSpan() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.endPrevSpanFn != nil {
		c.endPrevSpanFn()
		c.endPrevSpanFn = nil
	}
}

func (c *Consumer) createMetrics() error {
	var err error

	c.durationMeasure, err = meter.Float64Histogram(
		"message.consumer.duration",
		metric.WithUnit("ms"),
		metric.WithDescription("Measures the duration of message consumer"),
	)
	if err != nil {
		return err
	}

	return nil
}
