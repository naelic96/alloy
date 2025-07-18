package awstagprocessor

import (
    "github.com/grafana/alloy/internal/runtime/component"
    "github.com/grafana/alloy/internal/runtime/component/otelcol"
    "github.com/grafana/alloy/internal/runtime/component/otelcol/processor"
)

func init() {
    component.RegisterProcessor("awstagprocessor", NewFactory())
}
