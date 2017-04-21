package actions

import (
	"errors"

	"github.com/eagerio/Stout/src/actions/remotelogic"
	"github.com/eagerio/Stout/src/providermgmt"
	"github.com/eagerio/Stout/src/providers"
	"github.com/eagerio/Stout/src/types"
)

func Rollback(g types.GlobalFlags, r types.RollbackFlags) error {
	if g.FS == "" {
		return errors.New("The --fs flag is required for the `rollback` command")
	}
	if r.Version == "" {
		return errors.New("The --version flag is required for the `rollback` command")
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

	return remotelogic.Rollback(fsFuncs, g, r)
}
