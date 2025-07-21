package awstagprocessor

import (
	"context"
	"time"

	"github.com/grafana/alloy/internal/component"
	"github.com/grafana/alloy/internal/component/otelcol"
	"github.com/grafana/alloy/internal/component/otelcol/processor"
	"github.com/grafana/alloy/internal/featuregate"

	"go.opentelemetry.io/collector/component/componenthelper"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumerhelper"
)

const (
	TypeStr = "awstagprocessor"
)

func init() {
	processor.Register(TypeStr, NewFactory())
}

// NewFactory crea un processor Alloy compatibile
func NewFactory() otelcol.Factory {
	return otelcol.NewProcessorFactory(
		TypeStr,
		createDefaultConfig,
		otelcol.WithTracesProcessor(createTracesProcessor, component.StabilityAlpha),
		otelcol.WithLogsProcessor(createLogsProcessor, component.StabilityAlpha),
		otelcol.WithMetricsProcessor(createMetricsProcessor, component.StabilityAlpha),
	)
}

// Configurazione di default
func createDefaultConfig() component.Config {
	return &Config{
		ProcessorSettings: otelcol.NewProcessorSettings(component.NewID(TypeStr)),
		TTL:               6 * time.Hour,
	}
}

func createTracesProcessor(
	ctx context.Context,
	set otelcol.ProcessorCreateSettings,
	cfg component.Config,
	next consumer.Traces,
) (component.TracesProcessor, error) {
	oCfg := cfg.(*Config)
	p, err := newAWSTagProcessor(set.Logger, oCfg, tracesSignal)
	if err != nil {
		return nil, err
	}
	return consumerhelper.NewTracesProcessor(
		ctx,
		set.TelemetrySettings,
		cfg,
		next,
		p.processTraces,
		consumerhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
}

func createLogsProcessor(
	ctx context.Context,
	set otelcol.ProcessorCreateSettings,
	cfg component.Config,
	next consumer.Logs,
) (component.LogsProcessor, error) {
	oCfg := cfg.(*Config)
	p, err := newAWSTagProcessor(set.Logger, oCfg, logsSignal)
	if err != nil {
		return nil, err
	}
	return consumerhelper.NewLogsProcessor(
		ctx,
		set.TelemetrySettings,
		cfg,
		next,
		p.processLogs,
		consumerhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
}

func createMetricsProcessor(
	ctx context.Context,
	set otelcol.ProcessorCreateSettings,
	cfg component.Config,
	next consumer.Metrics,
) (component.MetricsProcessor, error) {
	oCfg := cfg.(*Config)
	p, err := newAWSTagProcessor(set.Logger, oCfg, metricsSignal)
	if err != nil {
		return nil, err
	}
	return consumerhelper.NewMetricsProcessor(
		ctx,
		set.TelemetrySettings,
		cfg,
		next,
		p.processMetrics,
		consumerhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
}
