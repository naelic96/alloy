package awstagprocessor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/grafana/alloy/internal/component"
	"github.com/grafana/alloy/internal/component/otelcol"
	"github.com/grafana/alloy/internal/component/otelcol/internal/fanoutconsumer"
	"github.com/grafana/alloy/internal/component/otelcol/internal/interceptconsumer"
	"github.com/grafana/alloy/internal/component/otelcol/internal/lazyconsumer"
	"github.com/grafana/alloy/internal/service/livedebugging"
	"github.com/grafana/alloy/internal/runtime/logging/level"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type Component struct {
	logger             log.Logger
	opts               component.Options
	args               Arguments
	debugDataPublisher livedebugging.DebugDataPublisher

	mu             sync.Mutex
	traceConsumer  *Processor
	logsConsumer   *Processor
	metricConsumer *Processor
}

var (
	_ component.Component     = (*Component)(nil)
	_ component.LiveDebugging = (*Component)(nil)
)

func buildProcessor(o component.Options, a component.Arguments) (component.Component, error) {
	args := a.(Arguments)

	debugPublisher, err := o.GetServiceData(livedebugging.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get livedebugging service: %w", err)
	}

	c := &Component{
		logger:             o.Logger,
		opts:               o,
		args:               args,
		debugDataPublisher: debugPublisher.(livedebugging.DebugDataPublisher),
	}

	if err := c.Update(a); err != nil {
		return nil, err
	}

	export := lazyconsumer.New(context.Background(), o.ID)
	export.SetConsumers(c.traceConsumer, c.metricConsumer, c.logsConsumer)
	o.OnStateChange(otelcol.ConsumerExports{Input: export})

	return c, nil
}

func (c *Component) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (c *Component) LiveDebugging() {}

func (c *Component) Update(newArgs component.Arguments) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	args := newArgs.(Arguments)

	c.args = args
	ttl := time.Duration(args.TTLHours) * time.Hour

	traceNext := args.Output.Traces
	traceFanout := fanoutconsumer.Traces(traceNext)

	logNext := args.Output.Logs
	logFanout := fanoutconsumer.Logs(logNext)

	metricNext := args.Output.Metrics
	metricFanout := fanoutconsumer.Metrics(metricNext)

	c.traceConsumer = NewProcessor(ttl, func(ctx context.Context, td ptrace.Traces) error {
		c.publishTraceDebugData(td, traceNext)
		return traceFanout.ConsumeTraces(ctx, td)
	})

	c.logsConsumer = NewProcessor(ttl, func(ctx context.Context, ld plog.Logs) error {
		// You could publish logs if needed
		return logFanout.ConsumeLogs(ctx, ld)
	})

	c.metricConsumer = NewProcessor(ttl, func(ctx context.Context, md pmetric.Metrics) error {
		// You could publish metrics if needed
		return metricFanout.ConsumeMetrics(ctx, md)
	})

	return nil
}

func (c *Component) publishTraceDebugData(td ptrace.Traces, next otelcol.TraceDataConsumer) {
	_ = livedebugging.PublishTracesIfActive(
		c.debugDataPublisher,
		c.opts.ID,
		td,
		otelcol.GetComponentMetadata(next),
	)
}
