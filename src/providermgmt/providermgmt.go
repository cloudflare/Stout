package providermgmt

import (
	"errors"
	"fmt"

	"github.com/eagerio/Stout/src/providers"
	"github.com/eagerio/Stout/src/providers/amazon"
	"github.com/eagerio/Stout/src/providers/cloudflare"
	"github.com/eagerio/Stout/src/providers/google"
	"github.com/eagerio/Stout/src/providers/none"
	"github.com/eagerio/Stout/src/types"
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
