package awstagprocessor

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

type tagCacheEntry struct {
	Tags      map[string]string
	Timestamp time.Time
}

type AWSTagEnricher struct {
	sync.RWMutex
	client *resourcegroupstaggingapi.ResourceGroupsTaggingAPI
	cache  map[string]tagCacheEntry
	ttl    time.Duration
}

func NewAWSTagEnricher(ttl time.Duration) (*AWSTagEnricher, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})
	if err != nil {
		return nil, err
	}

	client := resourcegroupstaggingapi.New(sess)

	return &AWSTagEnricher{
		client: client,
		cache:  make(map[string]tagCacheEntry),
		ttl:    ttl,
	}, nil
}

func (e *AWSTagEnricher) GetTags(ctx context.Context, arn string) (map[string]string, error) {
	e.RLock()
	entry, found := e.cache[arn]
	e.RUnlock()

	if found && time.Since(entry.Timestamp) < e.ttl {
		return entry.Tags, nil
	}

	// Make AWS API call to fetch tags
	input := &resourcegroupstaggingapi.GetResourcesInput{
		ResourceARNList: []*string{aws.String(arn)},
	}

	output, err := e.client.GetResourcesWithContext(ctx, input)
	if err != nil {
		return nil, err
	}

	tags := make(map[string]string)
	for _, resourceTagMapping := range output.ResourceTagMappingList {
		for _, tag := range resourceTagMapping.Tags {
			tags[*tag.Key] = *tag.Value
		}
	}

	// Cache the result
	e.Lock()
	e.cache[arn] = tagCacheEntry{
		Tags:      tags,
		Timestamp: time.Now(),
	}
	e.Unlock()

	return tags, nil
}

// ApplyTagsToAttributes attaches tags to the provided attribute map
func ApplyTagsToAttributes(tags map[string]string, attrs pcommon.Map) {
	for k, v := range tags {
		attrs.PutStr(k, v)
	}
}
