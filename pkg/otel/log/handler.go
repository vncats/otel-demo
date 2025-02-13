package log

import (
	"context"
	"log/slog"
	"runtime"

	slogctx "github.com/veqryn/slog-context"
	"go.opentelemetry.io/contrib/bridges/otelslog"
)

type Option func(*Handler)

func WithDebugLevel(enabled bool) Option {
	return func(h *Handler) {
		if enabled {
			h.level = slog.LevelDebug
		}
	}
}

type Handler struct {
	next  slog.Handler
	level slog.Level
}

var _ slog.Handler = &Handler{} // Assert conformance with interface

func NewHandler(name string, opts ...Option) *Handler {
	var next slog.Handler
	next = otelslog.NewHandler(name, otelslog.WithSource(true))
	next = slogctx.NewHandler(next, nil)

	handler := &Handler{
		next:  next,
		level: slog.LevelInfo,
	}
	for _, opt := range opts {
		opt(handler)
	}

	return handler
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h.next.WithAttrs(attrs)
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return h.next.WithGroup(name)
}

func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	if pc, _, _, ok := runtime.Caller(4); ok {
		r = r.Clone()
		r.PC = pc
	}

	return h.next.Handle(ctx, r)
}
