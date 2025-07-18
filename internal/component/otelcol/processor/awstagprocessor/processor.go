package awstagprocessor

import (
    "context"
    "fmt"

    "github.com/grafana/alloy/internal/runtime/component"
    "github.com/grafana/alloy/internal/runtime/component/otelcol"
    "go.opentelemetry.io/collector/component"
    "go.opentelemetry.io/collector/pdata/pcommon"
    "go.opentelemetry.io/collector/pdata/plog"
    "go.opentelemetry.io/collector/processor/processorhelper"
    "github.com/grafana/alloy/internal/runtime/component/otelcol/processor"
)

func NewFactory() component.ProcessorFactory {
    return processorthelper.NewFactory(
        "awstagprocessor",
        createDefaultConfig,
        processorthelper.WithLogs(createLogsProcessor),
    )
}

func createDefaultConfig() component.Config {
    return &Config{
        CacheFile: "/var/lib/alloy/aws_tag_cache.json",
        TTL:       21600, // 6 ore in secondi
    }
}

func createLogsProcessor(ctx context.Context, set component.ProcessorCreateSettings, cfg component.Config, next component.LogsConsumer) (component.LogsProcessor, error) {
    c := cfg.(*Config)
    cache, err := NewTagCache(c.CacheFile, c.TTL)
    if err != nil {
        return nil, err
    }
    client := NewAWSClient()

    return &processor{
        next:  next,
        cache: cache,
        aws:   client,
    }, nil
}

type processor struct {
    next  component.LogsConsumer
    cache *TagCache
    aws   *AWSClient
}

func (p *processor) Capabilities() component.ProcessorCapabilities {
    return component.ProcessorCapabilities{MutatesData: true}
}

func (p *processor) Start(ctx context.Context, host component.Host) error {
    return nil
}

func (p *processor) Shutdown(ctx context.Context) error {
    return p.cache.Save()
}

func (p *processor) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
    resLogs := ld.ResourceLogs()
    for i := 0; i < resLogs.Len(); i++ {
        rl := resLogs.At(i)
        attrs := rl.Resource().Attributes()

        arn, ok := attrs.Get("aws.arn")
        if !ok {
            continue
        }

        arnStr := arn.AsString()
        tags, found := p.cache.Get(arnStr)
        if !found {
            fetchedTags, err := p.aws.FetchTags(ctx, arnStr)
            if err != nil {
                continue
            }
            p.cache.Set(arnStr, fetchedTags)
            tags = fetchedTags
        }

        for k, v := range tags {
            attrs.PutStr(fmt.Sprintf("aws.tag.%s", k), v)
        }
    }
    return p.next.ConsumeLogs(ctx, ld)
}
