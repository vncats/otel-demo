package handler

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
)

type Server struct {
	app *httpkit.App
}

func NewServer(h *Handler, tc client.Client) *Server {
	app := httpkit.NewApp(httpkit.ServiceInfo{}, nil, httpkit.WithTracing())
	app.AddMiddleware(LogActivity(tc))
	app.SetupApplication()
	app.AddPublicRouteHandlers(initRoutes(h)...)

	return &Server{app: app}
}

func (s *Server) Start(ctx context.Context) error {
	server, err := s.app.NewHTTPServer(":8080", httpkit.ServerOption{})
	if err != nil {
		return err
	}

	fmt.Println("== Starting server at http://localhost:8080/movies. Press Ctrl+C to stop.")
	fmt.Println("== Jaeger UI: http://localhost:16686")
	fmt.Println("== Prometheus: http://localhost:9090")

	return server.Start(ctx)
}

type MiddlewareFn func(ctx *httpkit.RequestContext)

func (fn MiddlewareFn) Handle(ctx *httpkit.RequestContext) {
	fn(ctx)
}

func LogActivity(tc client.Client) MiddlewareFn {
	return func(ctx *httpkit.RequestContext) {
		_, _ = tc.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{
			ID:        uuid.NewString(),
			TaskQueue: workflow.TaskQueue,
		}, workflow.LogActivityWorkflow, container.Map{
			"user_agent": ctx.Request.Header.Get("User-Agent"),
			"route":      ctx.Route,
			"params":     ctx.Params,
		})

		ctx.Next()
	}
}
