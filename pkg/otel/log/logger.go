package log

import (
	"context"
	"log/slog"
	"sync/atomic"

	slogctx "github.com/veqryn/slog-context"
)

var globalLogger atomic.Value

func init() {
	globalLogger.Store(slog.Default())
}

func Logger() *slog.Logger {
	return globalLogger.Load().(*slog.Logger)
}

// Setup sets up the global logger with OTel logger provider
// If none is registered then default logger will be used
func Setup(name string, opts ...Option) {
	handler := NewHandler(name, opts...)
	logger := slog.New(handler)
	globalLogger.Store(logger)
}

func Info(ctx context.Context, msg string, args ...any) {
	Logger().InfoContext(ctx, msg, args...)
}

func Error(ctx context.Context, msg string, args ...any) {
	Logger().ErrorContext(ctx, msg, args...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	Logger().WarnContext(ctx, msg, args...)
}

func Debug(ctx context.Context, msg string, args ...any) {
	Logger().DebugContext(ctx, msg, args...)
}

func WithContext(ctx context.Context, args ...any) context.Context {
	return slogctx.Append(ctx, args...)
}
