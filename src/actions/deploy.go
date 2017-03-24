package actions

import (
	"errors"

	"github.com/eagerio/Stout/src/providers"
	"github.com/urfave/cli"
)

func Deploy(c *cli.Context) error {
	fsString := c.GlobalString("fs")

	if fsString == "" {
		return errors.New("The --fs flag and value are required for the `deploy` command")
	}

	err, fsProvider := providers.ValidateProviderType(fsString, providers.FS_PROVIDER_TYPE)
	if err != nil {
		return err
	}

	fsProviderTyped, _ := fsProvider.(providers.FSProvider)
	if err := fsProviderTyped.ValidateSettings(*c); err != nil {
		return err
	}

	if err := fsProviderTyped.DeployFS(*c); err != nil {
		return err
	}

	return nil
}
