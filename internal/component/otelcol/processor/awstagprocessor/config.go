package awstagprocessor

import (
	"time"

	"github.com/grafana/alloy/internal/component"
	"github.com/grafana/alloy/internal/component/otelcol"
)

// Config definisce la configurazione del processor
type Config struct {
	otelcol.ProcessorSettings `alloy:"squash"`

	// TTL indica per quanto tempo una entry cache rimane valida
	TTL time.Duration `alloy:"ttl,optional"`

	// UseRegionalSTS consente di usare endpoint STS regionali (utile in ambienti isolati)
	UseRegionalSTS bool `alloy:"use_regional_sts,optional"`

	// AllowedTags elenca i tag AWS da convertire in label (es. ["Environment", "App"])
	AllowedTags []string `alloy:"allowed_tags,optional"`

	// DisableTagFetching disattiva le chiamate API per fetch dei tag (solo per debug o test)
	DisableTagFetching bool `alloy:"disable_tag_fetching,optional"`
}
