package actions

import (
	"errors"

	"github.com/cloudflare/stout/src/actions/remotelogic"
	"github.com/cloudflare/stout/src/providermgmt"
	"github.com/cloudflare/stout/src/providers"
	"github.com/cloudflare/stout/src/types"
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
