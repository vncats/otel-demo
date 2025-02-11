package metric

import "go.opentelemetry.io/contrib/instrumentation/runtime"

// StartRuntime initializes reporting of go runtime metrics.
func StartRuntime() error {
	return runtime.Start()
}
