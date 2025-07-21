package awstagprocessor

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	"go.opentelemetry.io/collector/component"
)

// awsTagClient gestisce fetch e cache dei tag AWS
type awsTagClient struct {
	cfg         *Config
	logger      component.Logger
	apiClient   *resourcegroupstaggingapi.Client
	cache       map[string]cachedTags
	cacheMutex  sync.RWMutex
	cacheTTL    time.Duration
	lastRefresh map[string]time.Time
}

type cachedTags struct {
	Tags      map[string]string
	UpdatedAt time.Time
}

func newAWSTagClient(logger component.Logger, cfg *Config) (*awsTagClient, error) {
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(getAWSRegionFromEnv()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := resourcegroupstaggingapi.NewFromConfig(awsCfg)
	return &awsTagClient{
		cfg:         cfg,
		logger:      logger,
		apiClient:   client,
		cache:       make(map[string]cachedTags),
		cacheTTL:    cfg.TTL,
		lastRefresh: make(map[string]time.Time),
	}, nil
}

func (c *awsTagClient) GetTags(ctx context.Context, arnStr string) (map[string]string, error) {
	if c.cfg.DisableTagFetching {
		return nil, nil
	}

	// Controlla se è in cache e non è scaduto
	c.cacheMutex.RLock()
	cached, exists := c.cache[arnStr]
	if exists && time.Since(cached.UpdatedAt) < c.cacheTTL {
		c.cacheMutex.RUnlock()
		return cached.Tags, nil
	}
	c.cacheMutex.RUnlock()

	// Fetch da AWS
	c.logger.Debug(fmt.Sprintf("Fetching tags from AWS for ARN: %s", arnStr))
	parsedArn, err := arn.Parse(arnStr)
	if err != nil {
		return nil, fmt.Errorf("invalid ARN: %w", err)
	}

	resFilter := &resourcegroupstaggingapi.GetResourcesInput{
		ResourceARNList: []string{arnStr},
	}
	resp, err := c.apiClient.GetResources(ctx, resFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tags for %s: %w", arnStr, err)
	}

	tagMap := make(map[string]string)
	for _, resourceTagMapping := range resp.ResourceTagMappingList {
		for _, tag := range resourceTagMapping.Tags {
			if c.isTagAllowed(*tag.Key) {
				tagMap[*tag.Key] = *tag.Value
			}
		}
	}

	// Cache results
	c.cacheMutex.Lock()
	c.cache[arnStr] = cachedTags{
		Tags:      tagMap,
		UpdatedAt: time.Now(),
	}
	c.cacheMutex.Unlock()

	return tagMap, nil
}

func (c *awsTagClient) isTagAllowed(tagKey string) bool {
	if len(c.cfg.AllowedTags) == 0 {
		return true
	}
	for _, allowed := range c.cfg.AllowedTags {
		if strings.EqualFold(allowed, tagKey) {
			return true
		}
	}
	return false
}

// getAWSRegionFromEnv restituisce la regione da variabili d'ambiente
func getAWSRegionFromEnv() string {
	if region := os.Getenv("AWS_REGION"); region != "" {
		return region
	}
	if region := os.Getenv("AWS_DEFAULT_REGION"); region != "" {
		return region
	}
	return "us-east-1"
}
