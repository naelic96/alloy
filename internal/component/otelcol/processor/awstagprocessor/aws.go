package awstagprocessor

import (
    "context"
    "strings"

    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
)

type AWSClient struct {
    svc *resourcegroupstaggingapi.ResourceGroupsTaggingAPI
}

func NewAWSClient() *AWSClient {
    sess := session.Must(session.NewSession())
    svc := resourcegroupstaggingapi.New(sess)
    return &AWSClient{svc: svc}
}

func (c *AWSClient) FetchTags(ctx context.Context, arn string) (map[string]string, error) {
    input := &resourcegroupstaggingapi.GetResourcesInput{
        ResourceARNList: []*string{&arn},
    }

    out, err := c.svc.GetResourcesWithContext(ctx, input)
    if err != nil {
        return nil, err
    }

    tags := map[string]string{}
    for _, res := range out.ResourceTagMappingList {
        for _, tag := range res.Tags {
            tags[*tag.Key] = *tag.Value
        }
    }

    return tags, nil
}
