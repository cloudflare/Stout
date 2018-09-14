package actions

import (
	"errors"

	"github.com/cloudflare/stout/src/actions/remotelogic"
	"github.com/cloudflare/stout/src/providermgmt"
	"github.com/cloudflare/stout/src/providers"
	"github.com/cloudflare/stout/src/types"
)

func Deploy(g types.GlobalFlags, d types.DeployFlags) error {
	if g.FS == "" {
		return errors.New("The --fs flag is required for the `deploy` command")
	}

	err, fsProvider := providermgmt.ValidateProviderType(g.FS, types.FS_PROVIDER)
	if err != nil {
		return err
	}

	fsProviderTyped, _ := fsProvider.(providers.FSProvider)
	if err := fsProviderTyped.ValidateSettings(); err != nil {
		return err
	}

	fsFuncs, err := fsProviderTyped.FSProviderFuncs(g)
	if err != nil {
		return err
	}

	return remotelogic.Deploy(fsFuncs, g, d)
}
