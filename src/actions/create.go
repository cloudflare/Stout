package actions

import (
	"errors"
	"fmt"

	"github.com/eagerio/Stout/src/providers"
	"github.com/eagerio/Stout/src/providers/providermgmt"
)

func Create(g providers.GlobalFlags, c providers.CreateFlags) error {
	if g.FS == "" || g.CDN == "" || g.DNS == "" {
		return errors.New("The --dns, --fs, and --cdn flags and values are required for the `create` command")
	}

	err, fsProvider := providermgmt.ValidateProviderType(g.FS, providermgmt.FS_PROVIDER_TYPE)
	if err != nil {
		return err
	}
	err, cdnProvider := providermgmt.ValidateProviderType(g.CDN, providermgmt.CDN_PROVIDER_TYPE)
	if err != nil {
		return err
	}
	err, dnsProvider := providermgmt.ValidateProviderType(g.DNS, providermgmt.DNS_PROVIDER_TYPE)
	if err != nil {
		return err
	}

	fsProviderTyped, _ := fsProvider.(providers.FSProvider)
	if err := fsProviderTyped.ValidateSettings(); err != nil {
		return err
	}
	cdnProviderTyped, _ := cdnProvider.(providers.CDNProvider)
	if err := cdnProviderTyped.ValidateSettings(); err != nil {
		return err
	}
	dnsProviderTyped, _ := dnsProvider.(providers.DNSProvider)
	if err := dnsProviderTyped.ValidateSettings(); err != nil {
		return err
	}

	// during the create phase, the domain name for the cdn
	// needs to be provided to the dns
	fmt.Printf("Creating FS with %s...\n", fsProviderTyped.Name())
	fsDomain, err := fsProviderTyped.CreateFS(g, c)
	if err != nil {
		return err
	}
	fmt.Println()

	fmt.Printf("Creating CDN with %s...\n", cdnProviderTyped.Name())
	cdnDomain, err := cdnProviderTyped.CreateCDN(g, c, fsDomain)
	if err != nil {
		return err
	}
	fmt.Println()

	fmt.Printf("Creating DNS with %s...\n", dnsProviderTyped.Name())
	if err := dnsProviderTyped.CreateDNS(g, c, cdnDomain); err != nil {
		return err
	}
	fmt.Println()

	return nil
}
