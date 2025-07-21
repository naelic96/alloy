package awstagprocessor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processorthelper"
)

type awsTagProcessor struct {
	tagClient *resourcegroupstaggingapi.ResourceGroupsTaggingAPI
	ttl       time.Duration
	tagPrefix string

	mu    sync.Mutex
	cache map[string]cachedTags
}

type cachedTags struct {
	tags      map[string]string
	expiresAt time.Time
}

// newAWSTagProcessor creates the processor from Config
func newAWSTagProcessor(cfg *Config) (*awsTagProcessor, error) {
	awsCfg := aws.NewConfig()
	if cfg.Region != "" {
		awsCfg.Region = aws.String(cfg.Region)
	}
	if cfg.AssumeRoleARN != "" {
		// Role assumption logic can be added here
		// For simplicity, assuming credentials are available via env
		awsCfg.Credentials = credentials.NewEnvCredentials()
	}

	sess, err := session.NewSession(awsCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	client := resourcegroupstaggingapi.New(sess)

	return &awsTagProcessor{
		tagClient: client,
		ttl:       cfg.TTL,
		tagPrefix: cfg.TagPrefix,
		cache:     make(map[string]cachedTags),
	}, nil
}

// processTraces adds tags as resource attributes for traces
func (p *awsTagProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	rs := td.ResourceSpans()
	for i := 0; i < rs.Len(); i++ {
		res := rs.At(i).Resource()
		p.enrichResource(ctx, res.Attributes())
	}
	return td, nil
}

// processLogs adds tags as resource attributes for logs
func (p *awsTagProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	rs := ld.ResourceLogs()
	for i := 0; i < rs.Len(); i++ {
		res := rs.At(i).Resource()
		p.enrichResource(ctx, res.Attributes())
	}
	return ld, nil
}

// processMetrics adds tags as resource attributes for metrics
func (p *awsTagProcessor) processMetrics(ctx context.Context, md pmetric.Metrics) (pmetric.Metrics, error) {
	rs := md.ResourceMetrics()
	for i := 0; i < rs.Len(); i++ {
		res := rs.At(i).Resource()
		p.enrichResource(ctx, res.Attributes())
	}
	return md, nil
}

// enrichResource adds tags to the resource
func (p *awsTagProcessor) enrichResource(ctx context.Context, attrs pcommon.Map) {
	arnAttr, ok := attrs.Get("aws.arn")
	if !ok {
		arnAttr, ok = attrs.Get("cloud.resource_id")
		if !ok {
			return
		}
	}

	arn := arnAttr.Str()
	tags := p.getTagsForARN(ctx, arn)
	for k, v := range tags {
		attrs.PutStr(p.tagPrefix+k, v)
	}
}

// getTagsForARN returns tags from cache or AWS
func (p *awsTagProcessor) getTagsForARN(ctx context.Context, arn string) map[string]string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if cached, ok := p.cache[arn]; ok && time.Now().Before(cached.expiresAt) {
		return cached.tags
	}

	tags := p.fetchTagsFromAWS(ctx, arn)
	p.cache[arn] = cachedTags{
		tags:      tags,
		expiresAt: time.Now().Add(p.ttl),
	}
	return tags
}

// fetchTagsFromAWS queries AWS Resource Groups Tagging API for tags
func (p *awsTagProcessor) fetchTagsFromAWS(ctx context.Context, arn string) map[string]string {
	input := &resourcegroupstaggingapi.GetResourcesInput{
		ResourceARNList: []*string{aws.String(arn)},
	}
	output, err := p.tagClient.GetResourcesWithContext(ctx, input)
	if err != nil {
		return map[string]string{}
	}

	tags := map[string]string{}
	for _, resource := range output.ResourceTagMappingList {
		for _, tag := range resource.Tags {
			if tag.Key != nil && tag.Value != nil {
				tags[*tag.Key] = *tag.Value
			}
		}
	}
	return tags
}

// createProcessorFactory creates the processor factory for Alloy/OTel
func createProcessorFactory() component.ProcessorFactory {
	return processorthelper.NewFactory(
		"awstagprocessor",
		createDefaultConfig,
		processorthelper.WithTraces(newTraceProcessor),
		processorthelper.WithLogs(newLogProcessor),
		processorthelper.WithMetrics(newMetricProcessor),
	)
}

func newTraceProcessor(ctx context.Context, set component.ProcessorCreateSettings, cfg component.Config, next component.Traces) (component.Traces, error) {
	p, err := newAWSTagProcessor(cfg.(*Config))
	if err != nil {
		return nil, err
	}
	return processorthelper.NewTracesProcessor(ctx, set, cfg, next, p.processTraces)
}

func newLogProcessor(ctx context.Context, set component.ProcessorCreateSettings, cfg component.Config, next component.Logs) (component.Logs, error) {
	p, err := newAWSTagProcessor(cfg.(*Config))
	if err != nil {
		return nil, err
	}
	return processorthelper.NewLogsProcessor(ctx, set, cfg, next, p.processLogs)
}

func newMetricProcessor(ctx context.Context, set component.ProcessorCreateSettings, cfg component.Config, next component.Metrics) (component.Metrics, error) {
	p, err := newAWSTagProcessor(cfg.(*Config))
	if err != nil {
		return nil, err
	}
	return processorthelper.NewMetricsProcessor(ctx, set, cfg, next, p.processMetrics)
}
