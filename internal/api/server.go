package api

import (
	"context"
	"fmt"
	"github.com/vncats/otel-demo/pkg/otel/log"
	"github.com/vncats/otel-demo/pkg/prim"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"os"
	"os/signal"
	"time"
)

type Middleware func(http.Handler) http.Handler

type Server struct {
	server *http.Server
}

func NewServer(h IHandler) *Server {
	mux := http.NewServeMux()

	mux.Handle("/movies", newRouteHandler(
		h.GetMovies,
		TraceRequest("GET", "/movies"),
		TrackUserAction(h, "get_movies"),
	))

	mux.Handle("/movies/{id}/ratings/{score}", newRouteHandler(
		h.RateMovie,
		TraceRequest("GET", "/movies/{id}/ratings/{score}"),
		TrackUserAction(h, "rate_movies"),
	))

	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      mux,
	}
	return &Server{server: server}
}

func (s *Server) Start() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	srvErr := make(chan error, 1)
	go func() {
		srvErr <- s.server.ListenAndServe()
	}()

	// Wait for interruption.
	select {
	case <-srvErr:
		// Error when starting HTTP server.
		return
	case <-ctx.Done():
		stop()
	}

	_ = s.server.Shutdown(context.Background())
	log.Info(ctx, "server has shut down gracefully")
}

func newRouteHandler(handleFn func(ctx *RequestContext), middlewares ...Middleware) http.Handler {
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handleFn(&RequestContext{Writer: w, Request: r})
	})
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

func TraceRequest(method, route string) Middleware {
	return func(next http.Handler) http.Handler {
		handler := otelhttp.WithRouteTag(route, next)
		handler = otelhttp.NewHandler(next, fmt.Sprintf("%s %s", method, route))
		return handler
	}
}

func TrackUserAction(h IHandler, action string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			spanContext := trace.SpanContextFromContext(r.Context())
			payload := prim.Map{
				"user_id":    getUserID(r),
				"action":     action,
				"user_agent": r.Header.Get("User-Agent"),
			}
			go func() {
				newCtx, newSpan := otel.Tracer("middleware").Start(
					context.Background(), "track_user_action",
					trace.WithLinks(trace.Link{SpanContext: spanContext}),
				)
				defer newSpan.End()
				h.TrackUserAction(newCtx, payload)
			}()

			next.ServeHTTP(w, r)
		})
	}
}
