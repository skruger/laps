package dnsclient

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	r53types "github.com/aws/aws-sdk-go-v2/service/route53/types"

	"laps/config"
)

// UpdateRoute53AAAA updates (upserts) the AAAA record for the given
// fully-qualified hostname in the provided Route53 hosted zone. The function
// uses the default AWS configuration loading chain (environment, shared
// credentials, etc.).
//
// Parameters:
//   - ctx: context for the AWS call
//   - cfg: application config (unused for AWS creds; present for compatibility)
//   - fqdn: fully-qualified domain name to update (e.g. "host.example.com.")
//   - domain: domain name (unused if fqdn provided, kept for compatibility)
//   - ipv6: IPv6 address string (e.g. "2001:db8::1")
func UpdateRoute53AAAA(ctx context.Context, cfg *config.Config, fqdn, ipv6 string) error {
	if fqdn == "" {
		return fmt.Errorf("fqdn is required")
	}
	if ipv6 == "" {
		return fmt.Errorf("ipv6 address is required")
	}

	// Load AWS configuration (uses env, shared creds, etc.)
	awsCfg, err := awscfg.LoadDefaultConfig(ctx, awscfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AwsAccessKeyID, cfg.AwsSecretKey, "")))
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}
	awsCfg.Region = cfg.AwsRegion

	client := route53.NewFromConfig(awsCfg)

	// Prepare change batch to UPSERT the AAAA record
	rr := r53types.ResourceRecord{Value: aws.String(ipv6)}
	rrs := []r53types.ResourceRecord{rr}

	rrsSet := &r53types.ResourceRecordSet{
		Name:            aws.String(fqdn),
		Type:            r53types.RRTypeAaaa,
		TTL:             aws.Int64(300),
		ResourceRecords: rrs,
	}

	change := r53types.Change{
		Action:            r53types.ChangeActionUpsert,
		ResourceRecordSet: rrsSet,
	}

	input := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(cfg.R53ZoneID),
		ChangeBatch: &r53types.ChangeBatch{
			Changes: []r53types.Change{change},
		},
	}

	_, err = client.ChangeResourceRecordSets(ctx, input)
	if err != nil {
		return fmt.Errorf("route53 update failed: %w", err)
	}
	return nil
}
