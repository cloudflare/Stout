package actions

import (
	"errors"

	"github.com/eagerio/Stout/src/actions/remotelogic"
	"github.com/eagerio/Stout/src/providermgmt"
	"github.com/eagerio/Stout/src/providers"
	"github.com/eagerio/Stout/src/types"
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
