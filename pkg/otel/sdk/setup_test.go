package sdk

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	config "go.opentelemetry.io/contrib/config/v0.3.0"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	lognoop "go.opentelemetry.io/otel/log/noop"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)

func TestSetup(t *testing.T) {
	vTrue := true
	tests := []struct {
		name               string
		cfg                config.OpenTelemetryConfiguration
		wantTracerProvider any
		wantMeterProvider  any
		wantLoggerProvider any
		wantErr            error
		wantShutdownErr    error
	}{
		{
			name:               "no-configuration",
			wantTracerProvider: tracenoop.NewTracerProvider(),
			wantMeterProvider:  metricnoop.NewMeterProvider(),
			wantLoggerProvider: lognoop.NewLoggerProvider(),
		},
		{
			name: "with-configuration",
			cfg: config.OpenTelemetryConfiguration{
				TracerProvider: &config.TracerProvider{},
				MeterProvider:  &config.MeterProvider{},
				LoggerProvider: &config.LoggerProvider{},
			},
			wantTracerProvider: &trace.TracerProvider{},
			wantMeterProvider:  &metric.MeterProvider{},
			wantLoggerProvider: &log.LoggerProvider{},
		},
		{
			name: "with-sdk-disabled",
			cfg: config.OpenTelemetryConfiguration{
				Disabled:       &vTrue,
				TracerProvider: &config.TracerProvider{},
				MeterProvider:  &config.MeterProvider{},
				LoggerProvider: &config.LoggerProvider{},
			},
			wantTracerProvider: tracenoop.NewTracerProvider(),
			wantMeterProvider:  metricnoop.NewMeterProvider(),
			wantLoggerProvider: lognoop.NewLoggerProvider(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shutdownFn, err := Setup(context.Background(), OTelSDKConfig{OpenTelemetryConfiguration: tt.cfg})
			require.Equal(t, tt.wantErr, err)
			require.IsType(t, tt.wantTracerProvider, otel.GetTracerProvider())
			require.IsType(t, tt.wantMeterProvider, otel.GetMeterProvider())
			require.IsType(t, tt.wantLoggerProvider, global.GetLoggerProvider())
			require.Equal(t, "compositeTextMapPropagator", reflect.TypeOf(otel.GetTextMapPropagator()).Name())
			require.Equal(t, tt.wantShutdownErr, shutdownFn(context.Background()))
		})
	}
}

func TestSetupWithYAML(t *testing.T) {
	tests := []struct {
		name               string
		file               string
		wantTracerProvider any
		wantMeterProvider  any
		wantLoggerProvider any
		wantErr            error
		wantShutdownErr    error
	}{
		{
			name:               "valid_config",
			file:               "../testdata/sample.yaml",
			wantTracerProvider: &trace.TracerProvider{},
			wantMeterProvider:  &metric.MeterProvider{},
			wantLoggerProvider: &log.LoggerProvider{},
		},
		{
			name:               "valid_empty",
			file:               "../testdata/valid_empty.yaml",
			wantTracerProvider: tracenoop.NewTracerProvider(),
			wantMeterProvider:  metricnoop.NewMeterProvider(),
			wantLoggerProvider: lognoop.NewLoggerProvider(),
		},
		{
			name: "invalid_bool",
			file: "../testdata/invalid_bool.yaml",
			wantErr: errors.New(`yaml: unmarshal errors:
  line 2: cannot unmarshal !!str ` + "`notabool`" + ` into bool`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shutdownFn, err := SetupFromYAML(context.Background(), WithConfigFile(tt.file), WithEnabled(true))
			if tt.wantErr != nil {
				require.Equal(t, tt.wantErr.Error(), err.Error())
				return
			}
			require.NoError(t, err)
			require.IsType(t, tt.wantTracerProvider, otel.GetTracerProvider())
			require.IsType(t, tt.wantMeterProvider, otel.GetMeterProvider())
			require.IsType(t, tt.wantLoggerProvider, global.GetLoggerProvider())
			require.Equal(t, "compositeTextMapPropagator", reflect.TypeOf(otel.GetTextMapPropagator()).Name())
			require.Equal(t, tt.wantShutdownErr, shutdownFn(context.Background()))
		})
	}
}
