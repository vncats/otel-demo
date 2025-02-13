package tracing

import (
	"context"
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// WrapProducer wraps a kafka.Producer so that any produced messages are traced.
func WrapProducer(p *kafka.Producer, opts WrapOptions) *Producer {
	return &Producer{
		Producer:  p,
		spanAttrs: opts.SpanAttrs,
	}
}

type Producer struct {
	*kafka.Producer
	spanAttrs []attribute.KeyValue
}

// Produce calls the underlying Producer.Produce and traces the request.
func (p *Producer) Produce(msg *kafka.Message, deliveryChan chan kafka.Event) error {
	span := p.startSpan(msg)

	// if the user has selected a delivery channel, we will wrap it and
	// wait for the delivery event to finish the span
	if deliveryChan != nil {
		oldDeliveryChan := deliveryChan
		deliveryChan = make(chan kafka.Event)
		go func() {
			evt := <-deliveryChan
			if resMsg, ok := evt.(*kafka.Message); ok {
				if err := resMsg.TopicPartition.Error; err != nil {
					span.RecordError(resMsg.TopicPartition.Error)
					span.SetStatus(codes.Error, err.Error())
				}
			}
			span.End()
			oldDeliveryChan <- evt
		}()
	}

	err := p.Producer.Produce(msg, deliveryChan)

	// with no delivery channel or enqueue error, finish immediately
	if err != nil || deliveryChan == nil {
		span.RecordError(err)
		span.End()
	}

	return err
}

func (p *Producer) startSpan(msg *kafka.Message) trace.Span {
	carrier := NewMessageCarrier(msg)
	parentCtx := propagator.Extract(context.Background(), carrier)

	attrs := []attribute.KeyValue{
		semconv.MessagingOperationTypePublish,
		semconv.MessagingSystemKafka,
		semconv.MessagingDestinationName(*msg.TopicPartition.Topic),
		semconv.MessagingKafkaMessageKey(string(msg.Key)),
		semconv.MessagingMessageBodySize(getMsgSize(msg)),
	}
	attrs = append(attrs, p.spanAttrs...)

	spanCtx, span := tracer.Start(
		parentCtx,
		fmt.Sprintf("send %s", *msg.TopicPartition.Topic),
		trace.WithAttributes(attrs...),
		trace.WithSpanKind(trace.SpanKindProducer),
	)
	propagator.Inject(spanCtx, carrier)

	return span
}
