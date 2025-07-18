package awstagprocessor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-logfmt/logfmt"

	"github.com/grafana/alloy/internal/component"
	"go.opentelemetry.io/collector/featuregate"
	"github.com/grafana/alloy/internal/component/otelcol"
	"github.com/grafana/alloy/internal/component/otelcol/config"
	"github.com/grafana/alloy/internal/component/otelcol/exporter"
	"go.opentelemetry.io/collector/internal/fanoutconsumer"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configcompression"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processorhelper"
)
func init() {
component.Register(component.Registration{
Name:      "otelcol.processor.awstagprocessor",
Stability: featuregate.StabilityExperimental,
Args:      Arguments{},
Exports:   otelcol.ConsumerExports{},
Build: func(o component.Options, a component.Arguments) (component.Component, error) {
return New(o, a.(Arguments))
},
})
}

type Arguments struct {
Region string `alloy:"region,attr"`
Output *otelcol.ConsumerArguments `alloy:"output,block"`
}

func (args *Arguments) SetToDefault() {
args.Region = "us-west-2"
}

func (args *Arguments) Validate() error {
if args.Region == "" {
return fmt.Errorf("region is required")
}
return nil
}

type Component struct {
logger             log.Logger
debugDataPublisher livedebugging.DebugDataPublisher
opts               component.Options
args               Arguments
updateMut          sync.Mutex
}

func New(o component.Options, args Arguments) (*Component, error) {
debugDataPublisher, err := o.GetServiceData(livedebugging.ServiceName)
if err != nil {
return nil, err
}

next := args.Output.Traces
fanout := fanoutconsumer.Traces(next)

interceptor := interceptconsumer.Traces(fanout, func(ctx context.Context, td ptrace.Traces) error {
// Add AWS region as resource attribute
for i := 0; i < td.ResourceSpans().Len(); i++ {
res := td.ResourceSpans().At(i).Resource()
res.Attributes().PutStr("aws.region", args.Region)
}
livedebuggingpublisher.PublishTracesIfActive(debugDataPublisher, o.ID, td, otelcol.GetComponentMetadata(next))
return fanout.ConsumeTraces(ctx, td)
})

export := lazyconsumer.New(context.Background(), o.ID)
export.SetConsumers(interceptor, nil, nil)
o.OnStateChange(otelcol.ConsumerExports{Input: export})

return &Component{
logger:             o.Logger,
debugDataPublisher: debugDataPublisher.(livedebugging.DebugDataPublisher),
opts:               o,
args:               args,
}, nil
}

func (c *Component) Run(ctx context.Context) error {
<-ctx.Done()
return nil
}

func (c *Component) Update(newArgs component.Arguments) error {
c.updateMut.Lock()
defer c.updateMut.Unlock()
c.args = newArgs.(Arguments)
// Implement runtime changes if needed
return nil
}

func (c *Component) LiveDebugging() {} 