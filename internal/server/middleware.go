package server

import (
	"context"
	"fmt"
	"net/http"

	"go.opentelemetry.io/otel/propagation"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/vncats/otel-demo/pkg/prim"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

func TraceRequest(method, route string) Middleware {
	return func(next http.Handler) http.Handler {
		operationName := fmt.Sprintf("%s %s", method, route)
		attrs := []attribute.KeyValue{
			semconv.HTTPRoute(route),
			attribute.String("http.operation.name", operationName),
		}

		next = WithAttributes(next, attrs...)

		return otelhttp.NewHandler(next, operationName)
	}
}

func WithAttributes(h http.Handler, attrs ...attribute.KeyValue) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		span.SetAttributes(attrs...)

		labeler, _ := otelhttp.LabelerFromContext(r.Context())
		labeler.Add(attrs...)

		h.ServeHTTP(w, r)
	})
}

func TrackUserAction(h IHandler, action string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			go func() {
				// clone context
				carrier := propagation.MapCarrier{}
				otel.GetTextMapPropagator().Inject(r.Context(), carrier)
				newCtx := otel.GetTextMapPropagator().Extract(context.Background(), carrier)

				h.TrackUserAction(newCtx, prim.Map{
					"user_id":    getUserID(r),
					"request_id": getRequestID(r),
					"user_agent": r.Header.Get("User-Agent"),
					"action":     action,
				})
			}()
			next.ServeHTTP(w, r)
		})
	}
}
