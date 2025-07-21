package awstagprocessor

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"

	"go.opentelemetry.io/collector/component"
)

type signalType int

const (
	tracesSignal signalType = iota
	logsSignal
	metricsSignal
)

type awsTagProcessor struct {
	logger    component.Logger
	cfg       *Config
	client    *awsTagClient
	signal    signalType
	initOnce  sync.Once
	initError error
}

func newAWSTagProcessor(logger component.Logger, cfg *Config, signal signalType) (*awsTagProcessor, error) {
	client, err := newAWSTagClient(logger, cfg)
	if err != nil {
		return nil, err
	}

	return &awsTagProcessor{
		logger: logger,
		cfg:    cfg,
		client: client,
		signal: signal,
	}, nil
}

// ----------- TRACE -----------
func (p *awsTagProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	rs := td.ResourceSpans()
	for i := 0; i < rs.Len(); i++ {
		res := rs.At(i).Resource()
		p.enrichResource(ctx, res)
	}
	return td, nil
}

// ----------- LOG -----------
func (p *awsTagProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	rs := ld.ResourceLogs()
	for i := 0; i < rs.Len(); i++ {
		res := rs.At(i).Resource()
		p.enrichResource(ctx, res)
	}
	return ld, nil
}

// ----------- METRIC -----------
func (p *awsTagProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	rs := md.ResourceMetrics()
	for i := 0; i < rs.Len(); i++ {
		res := rs.At(i).Resource()
		p.enrichResource(ctx, res)
	}
	return md, nil
}

// ----------- COMMON -----------
func (p *awsTagProcessor) enrichResource(ctx context.Context, res pcommon.Resource) {
	arnAttr, exists := res.Attributes().Get("aws.arn")
	if !exists {
		return
	}

	arnStr := arnAttr.Str()
	tags, err := p.client.GetTags(ctx, arnStr)
	if err != nil {
		p.logger.Warn(fmt.Sprintf("Failed to fetch tags for ARN %s: %v", arnStr, err))
		return
	}

	for k, v := range tags {
		labelKey := sanitizeKey(fmt.Sprintf("aws.tag.%s", k))
		res.Attributes().PutStr(labelKey, v)
	}
}

func sanitizeKey(key string) string {
	// Sostituisce i caratteri non validi per le label di Prometheus/Loki/Tempo
	replacer := strings.NewReplacer(" ", "_", ".", "_", "/", "_", "-", "_")
	return replacer.Replace(strings.ToLower(key))
}
