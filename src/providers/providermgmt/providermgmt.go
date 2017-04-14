package providermgmt

import (
	"errors"
	"fmt"

	"github.com/eagerio/Stout/src/providers"
	"github.com/eagerio/Stout/src/providers/amazon"
	"github.com/eagerio/Stout/src/providers/cloudflare"
	"github.com/urfave/cli"
)

type providerConst string

const (
	DNS_PROVIDER_TYPE = providerConst("dns")
	FS_PROVIDER_TYPE  = providerConst("fs")
	CDN_PROVIDER_TYPE = providerConst("cdn")
)

var ProviderList = map[string]providers.ProviderClient{
	amazon.Client.Name():     &amazon.Client,
	cloudflare.Client.Name(): &cloudflare.Client,
}

func CreateCommandFlags() []cli.Flag {
	return commandFlags(true, true, true)
}

func DeployCommandFlags() []cli.Flag {
	return commandFlags(false, true, false)
}

func RollbackCommandFlags() []cli.Flag {
	return commandFlags(false, true, false)
}

func commandFlags(dns bool, fs bool, cdn bool) (flags []cli.Flag) {
	for _, provider := range ProviderList {
		addFlags := false

		if dns {
			if _, ok := provider.(providers.DNSProvider); ok {
				addFlags = true
			}
		}
		if fs {
			if _, ok := provider.(providers.FSProvider); ok {
				addFlags = true
			}
		}
		if cdn {
			if _, ok := provider.(providers.CDNProvider); ok {
				addFlags = true
			}
		}

		if addFlags {
			providerFlags := provider.Flags()
			//TODO: Add provider.Name() to the beginning of each flag
			flags = append(flags, providerFlags...)
		}
	}

	return
}

func ValidateProviderType(str string, providerType providerConst) (error, providers.ProviderClient) {
	possibleProvider, ok := ProviderList[str]
	if !ok {
		return errors.New(fmt.Sprintf("%q is not a supported provider (was attempted to be used as a %s provider)", str, providerType)), nil
	}

	if providerType == FS_PROVIDER_TYPE {
		if _, ok := possibleProvider.(providers.FSProvider); ok {
			return nil, possibleProvider
		}
	}
	if providerType == DNS_PROVIDER_TYPE {
		if _, ok := possibleProvider.(providers.DNSProvider); ok {
			return nil, possibleProvider
		}
	}
	if providerType == CDN_PROVIDER_TYPE {
		if _, ok := possibleProvider.(providers.CDNProvider); ok {
			return nil, possibleProvider
		}
	}

	return errors.New(fmt.Sprintf("%q is not a valid %s provider", str, providerType)), nil
}
