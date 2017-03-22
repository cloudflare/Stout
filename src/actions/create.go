package actions

import (
	"errors"
	"fmt"

	"github.com/eagerio/Stout/src/providers"
	"github.com/urfave/cli"
)

func Create(c *cli.Context) error {
	fsString := c.GlobalString("fs")
	cdnString := c.GlobalString("cdn")
	dnsString := c.GlobalString("dns")

	if fsString == "" || cdnString == "" || dnsString == "" {
		return errors.New("The --dns, --fs, and --cdn flags and values are required for the `create` command")
	}

	err, fsProvider := providers.ValidateProviderType(fsString, providers.FS_PROVIDER_TYPE)
	if err != nil {
		return err
	}
	err, cdnProvider := providers.ValidateProviderType(cdnString, providers.CDN_PROVIDER_TYPE)
	if err != nil {
		return err
	}
	err, dnsProvider := providers.ValidateProviderType(dnsString, providers.DNS_PROVIDER_TYPE)
	if err != nil {
		return err
	}

	fsProviderTyped, _ := fsProvider.(providers.FSProvider)
	if err := fsProviderTyped.ValidateSettings(*c); err != nil {
		return err
	}
	cdnProviderTyped, _ := cdnProvider.(providers.CDNProvider)
	if err := cdnProviderTyped.ValidateSettings(*c); err != nil {
		return err
	}
	dnsProviderTyped, _ := dnsProvider.(providers.DNSProvider)
	if err := dnsProviderTyped.ValidateSettings(*c); err != nil {
		return err
	}

	// during the create phase, the domain name for the cdn
	// needs to be provided to the dns
	if err := fsProviderTyped.CreateFS(*c); err != nil {
		return err
	}
	cdnDomain, err := cdnProviderTyped.CreateCDN(*c)
	if err != nil {
		return err
	}
	if err := dnsProviderTyped.CreateDNS(*c, cdnDomain); err != nil {
		return err
	}

	fmt.Println("You can begin deploying now, but it can take up to twenty minutes for your site to begin to work")
	fmt.Println("Depending on the configuration of your site, you might need to set the 'root', 'dest' or 'files' options to get your deploys working as you wish.  See the README for details.")
	fmt.Println("It's also a good idea to look into the 'env' option, as in real-world situations it usually makes sense to have a development and/or staging site for each of your production sites.")
	return nil
}
