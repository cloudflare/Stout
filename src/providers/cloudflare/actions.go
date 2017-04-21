package cloudflare

import (
	"errors"
	"fmt"

	"golang.org/x/net/publicsuffix"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/eagerio/Stout/src/types"
)

var skipDNS = false

func (a *client) CreateCDN(g types.GlobalFlags, c types.CreateFlags, fsDomain string) (string, error) {
	canCDN := g.DNS == a.Name()
	if !canCDN {
		return "", errors.New("Cloudflare cannot be used as a CDN without also being used as a DNS")
	}

	skipDNS = true
	return "", create(api, g.Domain, true, string(types.CNAME_RECORD), g.Domain, fsDomain)
}

func (a *client) CreateDNS(g types.GlobalFlags, c types.CreateFlags, cdnDomain string) error {
	if skipDNS {
		return nil
	}

	proxiable := g.CDN == a.Name()
	return create(api, g.Domain, proxiable, string(types.CNAME_RECORD), g.Domain, cdnDomain)
}

func (a *client) AddVerificationRecord(g types.GlobalFlags, c types.CreateFlags, recordType types.DNSRecordType, name string, value string) error {
	return create(api, g.Domain, false, string(recordType), name, value)
}

// One function to create CDN and DNS, since Cloudflare CDN depends on Cloudflare DNS
func create(api *cloudflare.API, domain string, useCDN bool, recordType string, name string, value string) error {
	zoneName, err := publicsuffix.EffectiveTLDPlusOne(domain)
	if err != nil {
		return err
	}

	zoneID, err := api.ZoneIDByName(zoneName)
	if err != nil {
		return err
	}

	// if it's not a TXT, then each name can only have one value
	if recordType != string(types.TXT_RECORD) {
		dnsRecords, err := api.DNSRecords(zoneID, cloudflare.DNSRecord{
			Type: recordType,
			Name: name,
		})
		if err != nil {
			return err
		}

		if len(dnsRecords) > 0 {
			for _, record := range dnsRecords {
				if record.Proxied != useCDN || record.Content != value {
					record.Proxied = useCDN
					record.Content = value
					return api.UpdateDNSRecord(zoneID, record.ID, record)
				}

				fmt.Println("Cloudflare already set up properly, skipping.")
				return nil
			}
		}
	}

	_, err = api.CreateDNSRecord(zoneID, cloudflare.DNSRecord{
		Type:    recordType,
		Name:    name,
		Content: value,
		Proxied: useCDN,
	})

	return err
}
