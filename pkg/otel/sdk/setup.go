package sdk

import (
	"context"
	config "go.opentelemetry.io/contrib/config/v0.3.0"
	"go.opentelemetry.io/contrib/processors/baggagecopy"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	"gopkg.in/yaml.v3"
	"os"
)

var defaultConfig = setupConfig{
	configFile: os.Getenv("OTEL_SDK_DEFAULT_CONFIG_FILE"),
	enabled:    os.Getenv("OTEL_SDK_ENABLED") == "true",
}

type setupConfig struct {
	configFile string
	enabled    bool
}

type Option func(*setupConfig)

func WithConfigFile(file string) Option {
	return func(c *setupConfig) {
		c.configFile = file
	}
}

func WithEnabled(enabled bool) Option {
	return func(c *setupConfig) {
		c.enabled = enabled
	}
}

type OTelSDKConfig struct {
	config.OpenTelemetryConfiguration `yaml:",inline"`
	ExtraConfig                       ExtraConfig `yaml:"extra_config"`
}

type ExtraConfig struct {
	Baggage []string `yaml:"baggage"`
}

// SetupFromYAML read OTel config from yaml file and setup SDK providers
// If filePath is empty, it will use the default config file path
func SetupFromYAML(ctx context.Context, opts ...Option) (func(context.Context) error, error) {
	c := defaultConfig
	for _, opt := range opts {
		opt(&c)
	}
	if !c.enabled {
		return func(context.Context) error { return nil }, nil
	}

	b, err := os.ReadFile(c.configFile)
	if err != nil {
		return nil, err
	}

	cfg, err := ParseYAML(b)
	if err != nil {
		return nil, err
	}

	return Setup(ctx, *cfg)
}

// Setup creates SDK providers based on the configuration model and set them as global providers
// Schema v0.3: https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/config/testdata/v0.3.yaml
func Setup(ctx context.Context, cfg OTelSDKConfig) (func(context.Context) error, error) {
	sdk, err := config.NewSDK(
		config.WithContext(ctx),
		config.WithOpenTelemetryConfiguration(cfg.OpenTelemetryConfiguration),
	)
	if err != nil {
		return nil, err
	}

	traceProvider := sdk.TracerProvider()
	meterProvider := sdk.MeterProvider()
	loggerProvider := sdk.LoggerProvider()

	// Register baggage processor
	if filter := baggageFilter(cfg.ExtraConfig.Baggage); filter != nil {
		if tp, ok := traceProvider.(*trace.TracerProvider); ok {
			tp.RegisterSpanProcessor(baggagecopy.NewSpanProcessor(filter))
		}
	}

	// Set global providers and propagators
	otel.SetTracerProvider(traceProvider)
	otel.SetMeterProvider(meterProvider)
	global.SetLoggerProvider(loggerProvider)
	otel.SetTextMapPropagator(defaultPropagator())

	return sdk.Shutdown, nil
}

// ParseYAML parses a YAML configuration file into an OTelSDKConfig.
func ParseYAML(file []byte) (*OTelSDKConfig, error) {
	var cfg OTelSDKConfig

	err := yaml.Unmarshal(file, &cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

// defaultPropagator returns the default TextMapPropagator
func defaultPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// baggageFilter returns a filter function for baggage members
func baggageFilter(keys []string) baggagecopy.Filter {
	if len(keys) == 0 {
		return nil
	}

	km := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		km[key] = struct{}{}
	}

	return func(member baggage.Member) bool {
		_, exists := km[member.Key()]
		return exists
	}
}
