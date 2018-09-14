package providermgmt

import (
	"errors"
	"fmt"

	"github.com/cloudflare/stout/src/providers"
	"github.com/cloudflare/stout/src/providers/amazon"
	"github.com/cloudflare/stout/src/providers/cloudflare"
	"github.com/cloudflare/stout/src/providers/google"
	"github.com/cloudflare/stout/src/providers/none"
	"github.com/cloudflare/stout/src/types"
)

var ProviderList = map[string]providers.ProviderClient{
	amazon.Client.Name():     &amazon.Client,
	cloudflare.Client.Name(): &cloudflare.Client,
	google.Client.Name():     &google.Client,
	none.Client.Name():       &none.Client,
}

func ValidateProviderType(str string, providerType types.Provider) (error, providers.ProviderClient) {
	possibleProvider, ok := ProviderList[str]
	if !ok {
		return errors.New(fmt.Sprintf("%q is not a supported provider (was attempted to be used as a %s provider)", str, providerType)), nil
	}

	if providerType == types.FS_PROVIDER {
		if _, ok := possibleProvider.(providers.FSProvider); ok {
			return nil, possibleProvider
		}
	}
	if providerType == types.DNS_PROVIDER {
		if _, ok := possibleProvider.(providers.DNSProvider); ok {
			return nil, possibleProvider
		}
	}
	if providerType == types.CDN_PROVIDER {
		if _, ok := possibleProvider.(providers.CDNProvider); ok {
			return nil, possibleProvider
		}
	}

	return errors.New(fmt.Sprintf("%q is not a valid %s provider", str, providerType)), nil
}
