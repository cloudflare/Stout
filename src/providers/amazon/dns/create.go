package dns

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/net/publicsuffix"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

// Add Route53 route
func CreateR53Route(r53Session *route53.Route53, domain string, cdnDomainName string) error {
	zone, err := getZone(r53Session, domain)
	if err != nil {
		return err
	}

	fmt.Printf("Adding %s to %s Route 53 zone\n", domain, *zone.Name)
	parts := strings.Split(*zone.Id, "/")
	idValue := parts[2]

	req, _ := r53Session.ChangeResourceRecordSetsRequest(&route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				&route53.Change{
					Action: aws.String("CREATE"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						AliasTarget: &route53.AliasTarget{
							HostedZoneId:         aws.String("Z2FDTNDATAQYW2"), //cloudfront distribution
							DNSName:              aws.String(cdnDomainName),
							EvaluateTargetHealth: aws.Bool(false),
						},
						Name: aws.String(domain),
						Type: aws.String("A"),
					},
				},
			},
		},
		HostedZoneId: aws.String(idValue),
	})
	err = req.Send()

	if err != nil {
		if strings.Contains(err.Error(), "it already exists") {
			fmt.Println("Existing route found, assuming it is correct")
			fmt.Printf("If you run into trouble, you may need to delete the %s route in Route53 and try again\n", domain)
			return nil
		}
		return err
	}

	return nil
}

func AddVerificationRecord(r53Session *route53.Route53, domain string, recordType string, name string, value string) error {
	zone, err := getZone(r53Session, domain)
	if err != nil {
		return err
	}

	fmt.Printf("Adding %s record with %s name and %s value %s Route 53 zone\n", recordType, name, value, *zone.Name)
	parts := strings.Split(*zone.Id, "/")
	idValue := parts[2]

	req, _ := r53Session.ChangeResourceRecordSetsRequest(&route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				&route53.Change{
					Action: aws.String("CREATE"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						ResourceRecords: []*route53.ResourceRecord{
							&route53.ResourceRecord{
								Value: aws.String(value),
							},
						},
						Name: aws.String(domain),
						Type: aws.String(recordType),
					},
				},
			},
		},
		HostedZoneId: aws.String(idValue),
	})

	return req.Send()
}

func getZone(r53Session *route53.Route53, domain string) (zone *route53.HostedZone, err error) {
	// get zone name
	zoneName, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return nil, err
	}

	zoneName = zoneName + "."

	// markers (for going past 100 items)
	var dnsName *string
	var hostedZoneID *string
	for {
		resp, err := r53Session.ListHostedZonesByName(&route53.ListHostedZonesByNameInput{
			DNSName:      dnsName,
			HostedZoneId: hostedZoneID,
			MaxItems:     aws.String("100"),
		})
		if err != nil {
			return nil, err
		}

		hostedZoneID = resp.NextHostedZoneId
		dnsName = resp.NextDNSName

		//find the first zone that matches the bucket zone name
		for _, z := range resp.HostedZones {
			if (*z.Name) == zoneName {
				zone = z
				break
			}
		}
		if zone != nil {
			fmt.Printf("Route53 hosted zone found for %s, continuing.\n", zoneName)
			break
		}

		if !*resp.IsTruncated {
			break
		}
	}

	if zone == nil {
		return nil, errors.New(fmt.Sprintf("A Route 53 hosted zone was not found for %s. This most likely means that you need to point your nameservers to Amazon or otherwise migrate your DNS. (http://docs.aws.amazon.com/Route53/latest/DeveloperGuide/MigratingDNS.html)", zoneName))
	}

	return zone, nil
}
