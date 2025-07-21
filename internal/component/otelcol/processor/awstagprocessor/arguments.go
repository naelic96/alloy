package awstagprocessor

import (
	"time"

	"github.com/grafana/alloy/internal/component"
	"github.com/grafana/alloy/internal/component/otelcol"
)

type Arguments struct {
	TTL time.Duration `river:"ttl,optional"`
}

// SetToDefault imposta i valori di default
func (args *Arguments) SetToDefault() {
	args.TTL = 6 * time.Hour
}

// Converti in config OTel compatibile
func (args *Arguments) Convert() (otelcol.Config, error) {
	return &Config{
		ProcessorSettings: otelcol.NewProcessorSettings(component.NewID(TypeStr)),
		TTL:               args.TTL,
	}, nil
}
