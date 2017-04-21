package actions

import (
	"errors"
	"fmt"
	"strings"

	"github.com/eagerio/Stout/src/providermgmt"
	"github.com/eagerio/Stout/src/providers"
	"github.com/eagerio/Stout/src/types"
)

func Create(g types.GlobalFlags, c types.CreateFlags) error {
	if g.FS == "" || g.CDN == "" || g.DNS == "" {
		return errors.New("The --dns, --fs, and --cdn flags are required for the `create` command")
	}

	err, fsProvider := providermgmt.ValidateProviderType(g.FS, types.FS_PROVIDER)
	if err != nil {
		return err
	}
	err, cdnProvider := providermgmt.ValidateProviderType(g.CDN, types.CDN_PROVIDER)
	if err != nil {
		return err
	}
	err, dnsProvider := providermgmt.ValidateProviderType(g.DNS, types.DNS_PROVIDER)
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

	if c.DomainValidationHelp {
		var input string
		fmt.Print("Enter record type to add for domain ownership validation (CNAME or TXT): ")
		fmt.Scanln(&input)

		input = strings.ToUpper(input)
		recordType := types.DNSRecordType(input)

		if recordType != types.CNAME_RECORD && recordType != types.TXT_RECORD {
			return errors.New("Invalid record type, must be CNAME or TXT.")
		}

		var name string
		if recordType == types.CNAME_RECORD {
			fmt.Println("Enter record subdomain to assign a value to.")
			fmt.Scanln(&name)
		} else {
			name = g.Domain
		}

		fmt.Println("Enter record value to assign (domain name for CNAME, or text string for TXT).")
		fmt.Scanln(&input)

		fmt.Println()
		fmt.Printf("Adding DNS based domain name verification with %s...\n", fsProviderTyped.Name())
		err := dnsProviderTyped.AddVerificationRecord(g, c, recordType, name, input)
		if err != nil {
			return err
		}
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
