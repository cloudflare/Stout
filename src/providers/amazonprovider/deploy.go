package amazonprovider

import (
	"github.com/eagerio/Stout/src/providers/amazonprovider/fs"
	"github.com/urfave/cli"
)

// Deploy a new version
func (a *client) DeployFS(c cli.Context) error {
	domain := c.GlobalString("domain")
	root := c.GlobalString("root")
	files := c.GlobalString("files")
	dest := c.GlobalString("dest")
	remote := c.GlobalString("remote")

	return fs.Deploy(s3Session, domain, root, files, dest, remote)
}
