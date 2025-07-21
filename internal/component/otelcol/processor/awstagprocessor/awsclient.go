package awstagprocessor

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
)

// TagCacheEntry holds the tags and the timestamp when they were last refreshed.
type TagCacheEntry struct {
	Tags      map[string]string
	Refreshed time.Time
}

// AWSClient wraps the AWS SDK client with a cache and mutex.
type AWSClient struct {
	client    *resourcegroupstaggingapi.Client
	cache     map[string]TagCacheEntry
	cacheTTL  time.Duration
	cacheLock sync.RWMutex
}

// NewAWSClient initializes the AWS SDK client using env variables.
func NewAWSClient(ctx context.Context, ttl time.Duration) (*AWSClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS config: %w", err)
	}

	client := resourcegroupstaggingapi.NewFromConfig(cfg)

	return &AWSClient{
		client:   client,
		cache:    make(map[string]TagCacheEntry),
		cacheTTL: ttl,
	}, nil
}

// GetTagsForResource returns the tags for the given ARN, with caching and TTL logic.
func (c *AWSClient) GetTagsForResource(ctx context.Context, arn string) (map[string]string, error) {
	c.cacheLock.RLock()
	entry, found := c.cache[arn]
	c.cacheLock.RUnlock()

	if found && time.Since(entry.Refreshed) < c.cacheTTL {
		return entry.Tags, nil
	}

	// TTL expired or not cached, refresh from AWS.
	output, err := c.client.GetResources(ctx, &resourcegroupstaggingapi.GetResourcesInput{
		ResourceARNList: []string{arn},
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching tags for ARN %s: %w", arn, err)
	}

	if len(output.ResourceTagMappingList) == 0 {
		return nil, fmt.Errorf("no tags found for ARN: %s", arn)
	}

	tags := make(map[string]string)
	for _, tag := range output.ResourceTagMappingList[0].Tags {
		tags[aws.ToString(tag.Key)] = aws.ToString(tag.Value)
	}

	c.cacheLock.Lock()
	c.cache[arn] = TagCacheEntry{
		Tags:      tags,
		Refreshed: time.Now(),
	}
	c.cacheLock.Unlock()

	return tags, nil
}

// InferARN attempts to derive an AWS ARN from log/trace/metric content.
func InferARN(attributes map[string]string) (string, bool) {
	// Common identifiers that could contain an ARN or partial info.
	candidates := []string{"aws.arn", "aws.log_group", "resource.arn", "logGroup", "log_group", "trace.id", "metric.name"}

	for _, key := range candidates {
		if val, ok := attributes[key]; ok && strings.HasPrefix(val, "arn:aws:") {
			return val, true
		}
	}

	return "", false
}
