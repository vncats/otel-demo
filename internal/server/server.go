package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/vncats/otel-demo/pkg/otel/log"
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
		TrackUserAction(h, "rate_movie"),
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

	fmt.Println("== Server is running at http://localhost:8080/movies")
	fmt.Println("== Traces: http://localhost:16686")
	fmt.Println("== Metrics: http://localhost:9090")

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
