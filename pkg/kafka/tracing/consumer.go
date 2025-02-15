package tracing

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/vncats/otel-demo/pkg/otel/sdk"

	"go.opentelemetry.io/otel/metric"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// WrapConsumer wraps a kafka.Consumer so that any consumed events are traced.
func WrapConsumer(c *kafka.Consumer, opts WrapOptions) (*Consumer, error) {
	consumer := &Consumer{
		Consumer:   c,
		attributes: opts.Attributes,
	}
	if err := consumer.createMetrics(); err != nil {
		return nil, err
	}

	return consumer, nil
}

type Consumer struct {
	*kafka.Consumer

	attributes        []attribute.KeyValue
	durationHistogram metric.Float64Histogram

	endPrevSpanFn func()
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
	carrier := NewMessageCarrier(msg)
	parentCtx := propagator.Extract(context.Background(), carrier)

	operationName := fmt.Sprintf("receive %s", *msg.TopicPartition.Topic)

	attrs := append(
		c.attributes,
		semconv.MessagingOperationName(operationName),
		semconv.MessagingOperationTypeReceive,
		semconv.MessagingSystemKafka,
		semconv.MessagingDestinationName(*msg.TopicPartition.Topic),
	)

	spanAttrs := append(
		attrs,
		semconv.MessagingKafkaMessageOffset(int(msg.TopicPartition.Offset)),
		semconv.MessagingKafkaMessageKey(string(msg.Key)),
		semconv.MessagingMessageID(strconv.FormatInt(int64(msg.TopicPartition.Offset), 10)),
		semconv.MessagingDestinationPartitionID(strconv.Itoa(int(msg.TopicPartition.Partition))),
		semconv.MessagingMessageBodySize(getMsgSize(msg)),
	)

	startTime := time.Now()
	spanCtx, span := tracer.Start(
		parentCtx,
		operationName,
		trace.WithNewRoot(),
		trace.WithTimestamp(startTime),
		trace.WithAttributes(spanAttrs...),
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithLinks(trace.Link{SpanContext: trace.SpanContextFromContext(parentCtx)}),
	)
	propagator.Inject(spanCtx, carrier)

	c.endPrevSpanFn = func() {
		c.recordMetrics(spanCtx, startTime, metric.WithAttributes(attrs...))
		span.End()
	}
}

func (c *Consumer) endPrevSpan() {
	if c.endPrevSpanFn != nil {
		c.endPrevSpanFn()
		c.endPrevSpanFn = nil
	}
}

func (c *Consumer) createMetrics() error {
	var err error

	c.durationHistogram, err = meter.Float64Histogram(
		semconv.MessagingProcessDurationName,
		metric.WithUnit(semconv.MessagingProcessDurationUnit),
		metric.WithDescription(semconv.MessagingProcessDurationDescription),
		metric.WithExplicitBucketBoundaries(sdk.HistogramBoundariesSeconds()...),
	)
	if err != nil {
		return err
	}

	return nil
}

func (c *Consumer) recordMetrics(ctx context.Context, startTime time.Time, opts ...metric.RecordOption) {
	c.durationHistogram.Record(ctx, time.Since(startTime).Seconds(), opts...)
}
