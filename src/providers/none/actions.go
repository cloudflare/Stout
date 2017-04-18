package none

import (
	"github.com/eagerio/Stout/src/providers"
)

func (cl *client) CreateCDN(g providers.GlobalFlags, c providers.CreateFlags, fsDomain string) (string, error) {
	return fsDomain, nil
}
