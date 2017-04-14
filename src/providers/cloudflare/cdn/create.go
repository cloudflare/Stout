package cdn

import (
	"fmt"

	"golang.org/x/net/publicsuffix"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

func Create(api *cloudflare.API, domain string, fsDomain string) error {
	zoneName, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return err
	}

	zoneID, err := api.ZoneIDByName(zoneName)
	if err != nil {
		return err
	}

	dnsRecords, err := api.DNSRecords(zoneID, cloudflare.DNSRecord{
		Type:    "CNAME",
		Name:    domain,
		Content: fsDomain,
	})
	if err != nil {
		return err
	}

	if len(dnsRecords) > 0 {
		for _, record := range dnsRecords {
			if !record.Proxiable {
				record.Proxiable = true
				err := api.UpdateDNSRecord(zoneID, record.ID, record)
				return err
			} else {
				fmt.Println("Cloudflare already set up properly, skipping.")
			}
			return nil
		}
	}

	_, err = api.CreateDNSRecord(zoneID, cloudflare.DNSRecord{
		Type:      "CNAME",
		Name:      domain,
		Content:   fsDomain,
		Proxiable: true,
	})
	if err != nil {
		return err
	}

	fmt.Println("Change other Cloudflare settings at https://www.cloudflare.com/a/overview")
	return nil
}
