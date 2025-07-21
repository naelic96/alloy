package awstagprocessor

import (
	"fmt"
	"strings"
)

func ExtractARNFromResource(resourceAttributes map[string]string) (string, error) {
	// Proviamo a ricavare un ARN da attributi noti
	if arn, ok := resourceAttributes["aws.arn"]; ok {
		return arn, nil
	}

	// Costruzione manuale (esempio: da LogGroupName + Region + AccountID + Service)
	region, regionOk := resourceAttributes["aws.region"]
	account, accountOk := resourceAttributes["aws.account_id"]
	resourceType, typeOk := resourceAttributes["aws.resource_type"]
	resourceName, nameOk := resourceAttributes["aws.resource_name"]

	if regionOk && accountOk && typeOk && nameOk {
		return fmt.Sprintf("arn:aws:%s:%s:%s:%s", resourceType, region, account, resourceName), nil
	}

	return "", fmt.Errorf("insufficient data to extract ARN")
}

func ExtractAttributesAsMap(attributes map[string]any) map[string]string {
	out := make(map[string]string)
	for k, v := range attributes {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

// NormalizeARN ensures we have a valid, lowercase, trimmed ARN
func NormalizeARN(arn string) string {
	return strings.TrimSpace(strings.ToLower(arn))
}
