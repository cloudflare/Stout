package providermgmt

import (
	"strings"

	"github.com/cloudflare/stout/src/providers"
	"github.com/cloudflare/stout/src/utils"
	"github.com/urfave/cli"
)

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

			flags = append(flags, utils.TitleFlag(strings.ToUpper(provider.Name())+" PROVIDER FLAGS:"))
			flags = append(flags, providerFlags...)
		}
	}

	return flags
}
