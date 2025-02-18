package log

import (
	"context"
	"log/slog"
	"runtime"

	"go.opentelemetry.io/otel/baggage"

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

func WithBaggageFilter(filter func(baggage.Member) bool) Option {
	return func(h *Handler) {
		h.filter = filter
	}
}

type Handler struct {
	next   slog.Handler
	level  slog.Level
	filter func(member baggage.Member) bool
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

	if h.filter != nil {
		r.AddAttrs(h.baggageAttributes(ctx)...)
	}

	return h.next.Handle(ctx, r)
}

func (h *Handler) baggageAttributes(ctx context.Context) []slog.Attr {
	var attrs []slog.Attr
	for _, member := range baggage.FromContext(ctx).Members() {
		if h.filter(member) {
			attrs = append(attrs, slog.String(member.Key(), member.Value()))
		}
	}

	return attrs
}
