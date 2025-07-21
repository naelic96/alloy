package awstagprocessor

import (
	"time"

	"go.opentelemetry.io/collector/config"
)

// Config defines configuration for the AWS tag processor.
type Config struct {
	config.ProcessorSettings `mapstructure:",squash"`

	// TTL defines how long to cache AWS tags for a resource before refreshing them.
	TTL time.Duration `mapstructure:"ttl"`

	// TagPrefix allows you to prefix added tag keys to avoid collisions.
	TagPrefix string `mapstructure:"tag_prefix"`

	// Region (optional) to override AWS region auto-detection.
	Region string `mapstructure:"region"`

	// AssumeRoleARN (optional) if you need to assume a specific role to fetch tags.
	AssumeRoleARN string `mapstructure:"assume_role_arn"`
}
