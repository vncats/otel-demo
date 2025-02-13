package message

import (
	"context"
	ckafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/vncats/otel-demo/pkg/kafka/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const defaultTracerName = "github.com/vncats/otel-demo/message"

var (
	propagator = otel.GetTextMapPropagator()
	tracer     = otel.Tracer(defaultTracerName)
)

func startSpan(msg *ckafka.Message, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	spanCtx := propagator.Extract(context.Background(), tracing.NewMessageCarrier(msg))
	return tracer.Start(spanCtx, name, opts...)
}
