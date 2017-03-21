package providers

import (
	"errors"
	"fmt"

	"github.com/eagerio/Stout/src/providers/amazonprovider"
	"github.com/urfave/cli"
)

type providerConst string

const (
	DNS_PROVIDER_TYPE = providerConst("dns")
	FS_PROVIDER_TYPE  = providerConst("fs")
	CDN_PROVIDER_TYPE = providerConst("cdn")
)

var providerList = map[string]ProviderClient{
	amazonprovider.Client.Name(): &amazonprovider.Client,
}

func CommandFlags(dns bool, fs bool, cdn bool) (flags []cli.Flag) {
	for _, provider := range providerList {
		addFlags := false

		if dns {
			if _, ok := provider.(DNSProvider); ok {
				addFlags = true
			}
		}
		if fs {
			if _, ok := provider.(FSProvider); ok {
				addFlags = true
			}
		}
		if cdn {
			if _, ok := provider.(CDNProvider); ok {
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

func ValidateProviderType(str string, providerType providerConst) (error, ProviderClient) {
	possibleProvider, ok := providerList[str]
	if !ok {
		return errors.New(fmt.Sprintf("%q is not a supported provider (was attempted to be used as a %s provider)", str, providerType)), nil
	}

	switch possibleProvider.(type) {
	case DNSProvider:
		if providerType == DNS_PROVIDER_TYPE {
			if _, ok := possibleProvider.(DNSProvider); ok {
				return nil, possibleProvider
			}
		}
	case FSProvider:
		if providerType == FS_PROVIDER_TYPE {
			if _, ok := possibleProvider.(FSProvider); ok {
				return nil, possibleProvider
			}
		}
	case CDNProvider:
		if providerType == CDN_PROVIDER_TYPE {
			if _, ok := possibleProvider.(CDNProvider); ok {
				return nil, possibleProvider
			}
		}
	}

	return errors.New(fmt.Sprintf("%q is not a valid %s provider", str, providerType)), nil
}
