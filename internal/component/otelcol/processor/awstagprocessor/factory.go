package awstagprocessor

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/processor"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertypes"
	"go.opentelemetry.io/collector/processor/processormetadata"
	"go.opentelemetry.io/collector/processor/processorfactory"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

const (
	// TypeStr is the unique identifier for this processor.
	TypeStr = "awstagprocessor"
)

// NewFactory returns a new processor factory for the awstagprocessor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		TypeStr,
		createDefaultConfig,
		processor.WithTraces(createTracesProcessor, component.StabilityAlpha),
		processor.WithLogs(createLogsProcessor, component.StabilityAlpha),
		processor.WithMetrics(createMetricsProcessor, component.StabilityAlpha),
	)
}

// createDefaultConfig returns a default configuration for this processor.
func createDefaultConfig() config.Processor {
	return &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewComponentID(TypeStr)),
		TTL:               6 * time.Hour,
	}
}

// createTracesProcessor sets up the processor for traces.
func createTracesProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg config.Processor,
	next consumer.Traces,
) (processor.Traces, error) {
	oCfg := cfg.(*Config)
	p, err := newAWSTagProcessor(set.Logger, oCfg, tracesSignal)
	if err != nil {
		return nil, err
	}
	return processorhelper.NewTracesProcessor(
		ctx,
		set,
		cfg,
		next,
		p.processTraces,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
}

// createLogsProcessor sets up the processor for logs.
func createLogsProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg config.Processor,
	next consumer.Logs,
) (processor.Logs, error) {
	oCfg := cfg.(*Config)
	p, err := newAWSTagProcessor(set.Logger, oCfg, logsSignal)
	if err != nil {
		return nil, err
	}
	return processorhelper.NewLogsProcessor(
		ctx,
		set,
		cfg,
		next,
		p.processLogs,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
}

// createMetricsProcessor sets up the processor for metrics.
func createMetricsProcessor(
	ctx context.Context,
	set processor.CreateSettings,
	cfg config.Processor,
	next consumer.Metrics,
) (processor.Metrics, error) {
	oCfg := cfg.(*Config)
	p, err := newAWSTagProcessor(set.Logger, oCfg, metricsSignal)
	if err != nil {
		return nil, err
	}
	return processorhelper.NewMetricsProcessor(
		ctx,
		set,
		cfg,
		next,
		p.processMetrics,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
}
