package log

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// MockLogExporter is a mock implementation of LogExporter for testing
type MockLogExporter struct {
	logs []sdklog.Record
}

func NewMockLogExporter() *MockLogExporter {
	return &MockLogExporter{
		logs: make([]sdklog.Record, 0),
	}
}

func (m *MockLogExporter) Export(_ context.Context, logs []sdklog.Record) error {
	m.logs = append(m.logs, logs...)
	return nil
}

func (m *MockLogExporter) Shutdown(_ context.Context) error {
	return nil
}

func (m *MockLogExporter) ForceFlush(_ context.Context) error {
	m.logs = make([]sdklog.Record, 0)
	return nil
}

func getAttrs(record sdklog.Record, names ...string) map[string]log.Value {
	attrs := make(map[string]log.Value)
	record.WalkAttributes(func(kv log.KeyValue) bool {
		if slices.Contains(names, kv.Key) {
			attrs[kv.Key] = kv.Value
		}
		return true
	})
	return attrs
}

func TestLogger(t *testing.T) {
	logExporter := NewMockLogExporter()
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(logExporter)),
	)
	global.SetLoggerProvider(loggerProvider)
	Setup("test")

	ctx := context.Background()
	requestID := uuid.NewString()
	ctx = WithContext(ctx, "request_id", requestID)

	Info(ctx, "inform sth", "key", "value", "foo", 123, "bar", false)
	Error(ctx, "failure sth", "error", errors.New("some error"))
	Warn(ctx, "warning sth", "warn", "this is a warning")
	Debug(ctx, "debug sth", "debug", "this is a debug")

	logs := logExporter.logs
	require.Len(t, logs, 3)

	infoLog := getAttrs(logs[0], "request_id", "code.filepath", "key", "foo", "bar")
	require.Equal(t, map[string]interface{}{
		"request_id": requestID,
		"key":        "value",
		"foo":        int64(123),
		"bar":        false,
	}, map[string]interface{}{
		"request_id": infoLog["request_id"].AsString(),
		"key":        infoLog["key"].AsString(),
		"foo":        infoLog["foo"].AsInt64(),
		"bar":        infoLog["bar"].AsBool(),
	})
	require.Contains(t, infoLog["code.filepath"].AsString(), "logger_test.go")

	errLog := getAttrs(logs[1], "request_id", "error")
	require.Equal(t, map[string]interface{}{
		"request_id": requestID,
		"error":      "some error",
	}, map[string]interface{}{
		"request_id": errLog["request_id"].AsString(),
		"error":      errLog["error"].AsString(),
	})

	warnLog := getAttrs(logs[2], "request_id", "warn")
	require.Equal(t, map[string]interface{}{
		"request_id": requestID,
		"warn":       "this is a warning",
	}, map[string]interface{}{
		"request_id": warnLog["request_id"].AsString(),
		"warn":       warnLog["warn"].AsString(),
	})

	Setup("test", WithDebugLevel(true))

	Debug(ctx, "debug sth", "debug", "this is a debug")
	require.Len(t, logExporter.logs, 4)

	debugLog := getAttrs(logExporter.logs[3], "request_id", "debug")
	require.Equal(t, map[string]interface{}{
		"request_id": requestID,
		"debug":      "this is a debug",
	}, map[string]interface{}{
		"request_id": debugLog["request_id"].AsString(),
		"debug":      debugLog["debug"].AsString(),
	})
}
