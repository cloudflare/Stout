package none

import "github.com/eagerio/Stout/src/types"

func (cl *client) CreateCDN(g types.GlobalFlags, c types.CreateFlags, fsDomain string) (string, error) {
	return fsDomain, nil
}
