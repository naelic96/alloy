package awstagprocessor

import (
    _ "github.com/grafana/alloy/internal/runtime/component"
)

func init() {
    component.RegisterProcessor("awstagprocessor", NewFactory())
}
