package awstagprocessor

import (
	"time"

	"github.com/grafana/alloy/internal/component"
	"github.com/grafana/alloy/internal/component/otelcol"
	"github.com/grafana/alloy/internal/featuregate"
	"github.com/grafana/alloy/syntax"
)

func init() {
	component.Register(component.Registration{
		Name:      "otelcol.processor.awstagprocessor",
		Stability: featuregate.StabilityExperimental,
		Args:      Arguments{},
		Exports:   otelcol.ConsumerExports{},
		Build: func(opts component.Options, args component.Arguments) (component.Component, error) {
			return New(opts, args.(Arguments))
		},
	})
}

// Arguments configures the awstagprocessor component.
type Arguments struct {
	// TTL defines the cache duration for AWS tags (e.g., "6h").
	TTL time.Duration `alloy:"ttl,attr,optional"`

	// Output is the next consumer that receives enriched telemetry data.
	Output *otelcol.ConsumerArguments `alloy:"output,block"`
}

// SetToDefault implements syntax.Defaulter.
func (args *Arguments) SetToDefault() {
	args.TTL = 6 * time.Hour
}

// Validate implements syntax.Validator.
func (args *Arguments) Validate() error {
	if args.Output == nil {
		return syntax.Error("output block is required")
	}
	if args.TTL <= 0 {
		return syntax.Error("ttl must be greater than 0")
	}
	return nil
}
