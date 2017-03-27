package actions

import (
	"errors"

	"github.com/eagerio/Stout/src/providers"
	"github.com/eagerio/Stout/src/types"
)

func Deploy(g types.GlobalFlags, d types.DeployFlags) error {
	if g.FS == "" {
		return errors.New("The --fs flag and value are required for the `deploy` command")
	}

	err, fsProvider := providers.ValidateProviderType(g.FS, providers.FS_PROVIDER_TYPE)
	if err != nil {
		return err
	}

	fsProviderTyped, _ := fsProvider.(providers.FSProvider)
	if err := fsProviderTyped.ValidateSettings(); err != nil {
		return err
	}

	if err := fsProviderTyped.DeployFS(g, d); err != nil {
		return err
	}

	return nil
}
