package dns

import (
	"fmt"

	"golang.org/x/net/publicsuffix"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

func Create(api *cloudflare.API, domain string, proxiable bool, cdnDomain string) error {
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
		Content: cdnDomain,
	})
	if err != nil {
		return err
	}

	if len(dnsRecords) > 0 {
		for _, record := range dnsRecords {
			if record.Proxiable != proxiable {
				record.Proxiable = proxiable
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
		Content:   cdnDomain,
		Proxiable: proxiable,
	})

	return err
}
