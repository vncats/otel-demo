package server

import (
	"context"
	"fmt"
	"github.com/vncats/otel-demo/pkg/prim"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
	"net/http"
)

func TraceRequest(method, route string) Middleware {
	return func(next http.Handler) http.Handler {
		handler := otelhttp.WithRouteTag(route, next)
		handler = otelhttp.NewHandler(next, fmt.Sprintf("%s %s", method, route))
		return handler
	}
}

func PopulateBaggage(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bag := baggage.FromContext(r.Context())
		bag = setMember(bag, "user_id", getUserID(r))
		bag = setMember(bag, "request_id", getRequestID(r))

		next.ServeHTTP(w, r.WithContext(baggage.ContextWithBaggage(r.Context(), bag)))
	})
}

func TrackUserAction(h IHandler, action string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload := prim.Map{
				"user_id":    getUserID(r),
				"request_id": getRequestID(r),
				"user_agent": r.Header.Get("User-Agent"),
				"action":     action,
			}
			go func() {
				spanCtx, span := otel.Tracer("middleware").Start(
					baggage.ContextWithBaggage(context.Background(), baggage.FromContext(r.Context())),
					"track_user_action",
					trace.WithLinks(trace.Link{
						SpanContext: trace.SpanContextFromContext(r.Context()),
					}),
				)
				defer span.End()
				h.TrackUserAction(spanCtx, payload)
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func setMember(bag baggage.Baggage, key, value string) baggage.Baggage {
	m, err := baggage.NewMember(key, value)
	if err != nil {
		return bag
	}

	b, err := bag.SetMember(m)
	if err != nil {
		return bag
	}

	return b
}
