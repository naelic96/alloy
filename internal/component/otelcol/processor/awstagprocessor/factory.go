package awstagprocessor

import (
	"context"
	"fmt"

	"github.com/grafana/alloy/internal/component"
	"github.com/grafana/alloy/internal/component/otelcol"
	"github.com/grafana/alloy/internal/featuregate"

	"go.opentelemetry.io/collector/component"
	colprocessor "go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerhelper"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Register the component with Alloy.
func init() {
	component.Register(component.Registration{
		Name:      "otelcol.processor.awstag",
		Stability: featuregate.StabilityExperimental,
		Args:      Arguments{},
		Exports:   otelcol.ConsumerExports{},
		Build: func(o component.Options, a component.Arguments) (component.Component, error) {
			return NewProcessor(o, a.(Arguments))
		},
	})
}

// NewProcessor builds the processor component.
func NewProcessor(o component.Options, args Arguments) (*Processor, error) {
	processor := &Processor{
		logger:  o.Logger,
		ttl:     args.TTL,
		args:    args,
		enricher: NewAWSTagEnricher(args.TTL, o.Logger),
	}

	export := otelcol.ConsumerExports{}
	export.Input = otelcol.Consumer{
		Traces:  processor,
		Logs:    processor,
		Metrics: processor,
	}
	o.OnStateChange(export)

	return processor, nil
}
