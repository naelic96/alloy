package awstagprocessor

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"
)

type tagInfo struct {
	tags      map[string]string
	expiresAt time.Time
}

type AWSEnricher struct {
	logger       *zap.Logger
	client       *resourcegroupstaggingapi.Client
	cfg          *Config
	cache        map[string]*tagInfo
	cacheMutex   sync.RWMutex
	defaultTTL   time.Duration
}

// Costruttore
func NewAWSEnricher(logger *zap.Logger, cfg *Config) *AWSEnricher {
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		logger.Fatal("unable to load AWS config", zap.Error(err))
	}

	client := resourcegroupstaggingapi.NewFromConfig(awsCfg)

	return &AWSEnricher{
		logger:     logger,
		client:     client,
		cfg:        cfg,
		cache:      make(map[string]*tagInfo),
		defaultTTL: cfg.TTL,
	}
}

// EnrichResourceAttributes inserisce i tag come attributi della risorsa
func (e *AWSEnricher) EnrichResourceAttributes(attrs pcommon.Map) {
	arn := extractARN(attrs)
	if arn == "" {
		e.logger.Debug("No ARN found in resource attributes")
		return
	}

	tags := e.getTagsForARN(arn)
	if tags == nil {
		e.logger.Debug("No tags found for ARN", zap.String("arn", arn))
		return
	}

	for k, v := range tags {
		attrs.PutStr("aws.tag."+k, v)
	}
}

// Estrae i tag, usando cache con TTL
func (e *AWSEnricher) getTagsForARN(arn string) map[string]string {
	e.cacheMutex.RLock()
	entry, found := e.cache[arn]
	e.cacheMutex.RUnlock()

	if found && time.Now().Before(entry.expiresAt) {
		return entry.tags
	}

	tags := e.fetchTags(arn)
	if tags == nil {
		return nil
	}

	e.cacheMutex.Lock()
	e.cache[arn] = &tagInfo{
		tags:      tags,
		expiresAt: time.Now().Add(e.defaultTTL),
	}
	e.cacheMutex.Unlock()

	return tags
}

// Chiamata all'AWS Resource Tagging API
func (e *AWSEnricher) fetchTags(arn string) map[string]string {
	out, err := e.client.GetResources(context.TODO(), &resourcegroupstaggingapi.GetResourcesInput{
		ResourceARNList: []string{arn},
	})
	if err != nil {
		e.logger.Warn("Failed to fetch tags from AWS", zap.Error(err), zap.String("arn", arn))
		return nil
	}
	if len(out.ResourceTagMappingList) == 0 {
		return nil
	}

	tagMap := make(map[string]string)
	for _, tag := range out.ResourceTagMappingList[0].Tags {
		tagMap[*tag.Key] = *tag.Value
	}

	return tagMap
}

// Estrae l'ARN da attributi noti
func extractARN(attrs pcommon.Map) string {
	keys := []string{
		"aws.arn", "cloud.resource_id", "cloud.arn",
	}
	for _, key := range keys {
		val, ok := attrs.Get(key)
		if ok {
			return val.Str()
		}
	}
	// Tentativo euristico
	if val, ok := attrs.Get("logGroup"); ok {
		return guessARNFromLogGroup(val.Str())
	}
	return ""
}

// Esempio di generazione ARN da logGroup
func guessARNFromLogGroup(logGroup string) string {
	region := os.Getenv("AWS_REGION")
	accountID := os.Getenv("AWS_ACCOUNT_ID")
	if region == "" || accountID == "" || logGroup == "" {
		return ""
	}
	return "arn:aws:logs:" + region + ":" + accountID + ":log-group:" + strings.TrimPrefix(logGroup, "/")
}
