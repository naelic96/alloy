package awstagprocessor

import (
	"errors"
	"time"

	"github.com/grafana/alloy/internal/component"
	"github.com/grafana/alloy/internal/component/otelcol"
	"github.com/grafana/alloy/syntax"
)

var (
	_ syntax.Validator = (*Arguments)(nil)
	_ component.ArgumentsConverter = (*Arguments)(nil)
)

// Arguments configures the awstagprocessor component.
type Arguments struct {
	// Output defines where to send processed telemetry.
	Output *otelcol.ConsumerArguments `alloy:"output,block"`

	// Refresh interval for AWS tag cache.
	TTL time.Duration `alloy:"ttl,attr,optional"`

	// Enable tag enrichment for traces.
	EnableTraces bool `alloy:"enable_traces,attr,optional"`

	// Enable tag enrichment for logs.
	EnableLogs bool `alloy:"enable_logs,attr,optional"`

	// Enable tag enrichment for metrics.
	EnableMetrics bool `alloy:"enable_metrics,attr,optional"`
}

// DefaultArguments returns the default processor settings.
func DefaultArguments() Arguments {
	return Arguments{
		TTL:           6 * time.Hour,
		EnableTraces:  true,
		EnableLogs:    true,
		EnableMetrics: true,
	}
}

// SetToDefault sets arguments to default values.
func (args *Arguments) SetToDefault() {
	*args = DefaultArguments()
}

// Validate ensures the arguments are correctly configured.
func (args Arguments) Validate() error {
	if args.Output == nil {
		return errors.New("output is required")
	}
	if !args.EnableTraces && !args.EnableLogs && !args.EnableMetrics {
		return errors.New("at least one of enable_traces, enable_logs, or enable_metrics must be true")
	}
	return nil
}

// Convert transforms Arguments into a component.Options-compatible struct.
func (args Arguments) Convert() (any, error) {
	return args, nil
}