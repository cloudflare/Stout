package cloudflare

import (
	"errors"
	"fmt"

	"golang.org/x/net/publicsuffix"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/eagerio/Stout/src/providers"
)

var skipDNS = false

// Set up Cloudflare CDN
func (a *client) CreateCDN(g providers.GlobalFlags, c providers.CreateFlags, fsDomain string) (string, error) {
	canCDN := g.DNS == a.Name()
	if !canCDN {
		return "", errors.New("Cloudflare cannot be used as a CDN without also being used as a DNS")
	}

	skipDNS = true
	return "", create(api, g.Domain, true, fsDomain)
}

// Set up Cloudflare DNS
func (a *client) CreateDNS(g providers.GlobalFlags, c providers.CreateFlags, cdnDomain string) error {
	if skipDNS {
		return nil
	}

	proxiable := g.CDN == a.Name()
	return create(api, g.Domain, proxiable, cdnDomain)
}

// One function to create CDN and DNS, since Cloudflare CDN depends on Cloudflare DNS
func create(api *cloudflare.API, domain string, useCDN bool, endDomain string) error {
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
		Content: endDomain,
	})
	if err != nil {
		return err
	}

	if len(dnsRecords) > 0 {
		for _, record := range dnsRecords {
			if record.Proxied != useCDN {
				record.Proxied = useCDN
				err := api.UpdateDNSRecord(zoneID, record.ID, record)
				return err
			} else {
				fmt.Println("Cloudflare already set up properly, skipping.")
			}
			return nil
		}
	}

	_, err = api.CreateDNSRecord(zoneID, cloudflare.DNSRecord{
		Type:    "CNAME",
		Name:    domain,
		Content: endDomain,
		Proxied: true,
	})
	if err != nil {
		return err
	}

	return nil
}
