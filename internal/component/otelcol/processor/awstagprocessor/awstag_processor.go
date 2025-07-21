package awstagprocessor

import (
	"context"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type signalType int

const (
	tracesSignal signalType = iota
	logsSignal
	metricsSignal
)

// awstagProcessor è il processore principale che applica i tag AWS alle risorse.
type awstagProcessor struct {
	logger     *zap.Logger
	cfg        *Config
	signalType signalType
	enricher   *AWSEnricher
}

// Costruttore del processor
func newAWSTagProcessor(logger *zap.Logger, cfg *Config, signal signalType) (*awstagProcessor, error) {
	enricher := NewAWSEnricher(logger, cfg)
	return &awstagProcessor{
		logger:     logger,
		cfg:        cfg,
		signalType: signal,
		enricher:   enricher,
	}, nil
}

// Processamento dei Traces
func (p *awstagProcessor) processTraces(_ context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	resourceSpans := td.ResourceSpans()
	for i := 0; i < resourceSpans.Len(); i++ {
		rs := resourceSpans.At(i)
		p.enricher.EnrichResourceAttributes(rs.Resource().Attributes())
	}
	return td, nil
}

// Processamento dei Logs
func (p *awstagProcessor) processLogs(_ context.Context, ld plog.Logs) (plog.Logs, error) {
	resourceLogs := ld.ResourceLogs()
	for i := 0; i < resourceLogs.Len(); i++ {
		rl := resourceLogs.At(i)
		p.enricher.EnrichResourceAttributes(rl.Resource().Attributes())
	}
	return ld, nil
}

// Processamento delle Metriche
func (p *awstagProcessor) processMetrics(_ context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	resourceMetrics := md.ResourceMetrics()
	for i := 0; i < resourceMetrics.Len(); i++ {
		rm := resourceMetrics.At(i)
		p.enricher.EnrichResourceAttributes(rm.Resource().Attributes())
	}
	return md, nil
}
