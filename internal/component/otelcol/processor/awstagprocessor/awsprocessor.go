package awstagprocessor

import (
	"context"
	"fmt"

	"github.com/go-kit/log"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// Processor holds the AWS client and config.
type Processor struct {
	logger    log.Logger
	awsClient *AWSClient
}

// NewProcessor creates a new processor with an AWS client and TTL.
func NewProcessor(ctx context.Context, logger log.Logger, ttlHours int) (*Processor, error) {
	client, err := NewAWSClient(ctx, hoursToDuration(ttlHours))
	if err != nil {
		return nil, err
	}

	return &Processor{
		logger:    logger,
		awsClient: client,
	}, nil
}

// ProcessTraces injects tags as resource attributes into traces.
func (p *Processor) ProcessTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	rs := td.ResourceSpans()

	for i := 0; i < rs.Len(); i++ {
		resource := rs.At(i).Resource()
		attributes := resource.Attributes().AsRaw()

		arn, found := InferARN(stringMap(attributes))
		if !found {
			continue
		}

		tags, err := p.awsClient.GetTagsForResource(ctx, arn)
		if err != nil {
			_ = logError(p.logger, "GetTagsForResource failed", arn, err)
			continue
		}

		for k, v := range tags {
			resource.Attributes().PutStr(fmt.Sprintf("aws.tag.%s", k), v)
		}
	}

	return td, nil
}

// ProcessLogs injects tags into logs as resource attributes.
func (p *Processor) ProcessLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	rs := ld.ResourceLogs()

	for i := 0; i < rs.Len(); i++ {
		resource := rs.At(i).Resource()
		attributes := resource.Attributes().AsRaw()

		arn, found := InferARN(stringMap(attributes))
		if !found {
			continue
		}

		tags, err := p.awsClient.GetTagsForResource(ctx, arn)
		if err != nil {
			_ = logError(p.logger, "GetTagsForResource failed", arn, err)
			continue
		}

		for k, v := range tags {
			resource.Attributes().PutStr(fmt.Sprintf("aws.tag.%s", k), v)
		}
	}

	return ld, nil
}

// ProcessMetrics injects tags into metrics as resource attributes.
func (p *Processor) ProcessMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	rs := md.ResourceMetrics()

	for i := 0; i < rs.Len(); i++ {
		resource := rs.At(i).Resource()
		attributes := resource.Attributes().AsRaw()

		arn, found := InferARN(stringMap(attributes))
		if !found {
			continue
		}

		tags, err := p.awsClient.GetTagsForResource(ctx, arn)
		if err != nil {
			_ = logError(p.logger, "GetTagsForResource failed", arn, err)
			continue
		}

		for k, v := range tags {
			resource.Attributes().PutStr(fmt.Sprintf("aws.tag.%s", k), v)
		}
	}

	return md, nil
}

// Utility to convert map[any]any to map[string]string.
func stringMap(raw map[string]any) map[string]string {
	result := make(map[string]string)
	for k, v := range raw {
		key, ok := k.(string)
		if !ok {
			continue
		}
		val, ok := v.(string)
		if !ok {
			continue
		}
		result[key] = val
	}
	return result
}

func hoursToDuration(hours int) time.Duration {
	return time.Duration(hours) * time.Hour
}

func logError(logger log.Logger, msg string, arn string, err error) error {
	_ = logger.Log("msg", msg, "arn", arn, "err", err)
	return err
}
