package dns

import (
	"fmt"
	"strings"

	"golang.org/x/net/publicsuffix"

	"github.com/zackbloom/goamz/route53"
)

// Add Route53 route
func UpdateR53Route(r53Session *route53.Route53, domain string, cdnDomainName string) error {
	//get zone name
	zoneName, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return err
	}

	zoneName = zoneName + "."

	resp, err := r53Session.ListHostedZonesByName(zoneName, "", 100)
	if err != nil {
		return err
	}

	if resp.IsTruncated {
		panic("More than 100 zones in the account")
	}

	//find the first zone that matches the bucket zone name
	var zone *route53.HostedZone
	for _, z := range resp.HostedZones {
		if z.Name == zoneName {
			zone = &z
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
			//the bucket is not a subdomain
			// TODO(renandincer): Simplify this, it might be confusing to a user
			fmt.Println("Since you are hosting the root of your domain, using an alternative DNS host is unfortunately not possible.")
			fmt.Println("If you wish to host your site at the root of your domain, you must switch your sites DNS to Amazon's Route 53 and retry this command.")
		}

		return nil
	}

	fmt.Printf("Adding %s to %s Route 53 zone\n", domain, zone.Name)
	parts := strings.Split(zone.Id, "/")
	idValue := parts[2]

	_, err = r53Session.ChangeResourceRecordSet(&route53.ChangeResourceRecordSetsRequest{
		Changes: []route53.Change{
			route53.Change{
				Action: "CREATE",
				Name:   domain,
				Type:   "A",
				AliasTarget: route53.AliasTarget{
					HostedZoneId:         "Z2FDTNDATAQYW2", //cloudfront distribution
					DNSName:              cdnDomainName,
					EvaluateTargetHealth: false,
				},
			},
		},
	}, idValue)

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
