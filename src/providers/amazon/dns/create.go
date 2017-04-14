package dns

import (
	"fmt"
	"strings"

	"golang.org/x/net/publicsuffix"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

// Add Route53 route
func UpdateR53Route(r53Session *route53.Route53, domain string, cdnDomainName string) error {
	// get zone name
	zoneName, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return err
	}

	zoneName = zoneName + "."

	var zone *route53.HostedZone

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
			return err
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
		fmt.Printf("A Route 53 hosted zone was not found for %s\n", zoneName)
		if zoneName != domain {
			// the bucket could not be found in route53 and is a subdomain
			fmt.Println("If you would like to use Route 53 to manage your DNS, create a zone for this domain, and update your registrar's configuration to point to the DNS servers Amazon provides and rerun this command.  Note that you must copy any existing DNS configuration you have to Route 53 if you do not wish existing services hosted on this domain to stop working.")
			fmt.Printf("If you would like to continue to use your existing DNS, create a CNAME record pointing %s to %s and the site setup will be finished.\n", domain, cdnDomainName)
		} else {
			// the bucket is not a subdomain
			// TODO(renandincer): Simplify this, it might be confusing to a user
			fmt.Println("Since you are hosting the root of your domain, using an alternative DNS host is unfortunately not possible.")
			fmt.Println("If you wish to host your site at the root of your domain, you must switch your sites DNS to Amazon's Route 53 and retry this command.")
		}

		return nil
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
