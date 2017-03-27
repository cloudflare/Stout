package providers

import (
	"github.com/eagerio/Stout/src/types"
	"github.com/urfave/cli"
)

type ProviderClient interface {
	Name() string
	Flags() []cli.Flag
	ValidateSettings() error
}

type DNSProvider interface {
	CreateDNS(g types.GlobalFlags, c types.CreateFlags, cdnDomain string) error

	Name() string
	Flags() []cli.Flag
	ValidateSettings() error
}

type FSProvider interface {
	CreateFS(g types.GlobalFlags, c types.CreateFlags) error
	DeployFS(g types.GlobalFlags, d types.DeployFlags) error
	RollbackFS(g types.GlobalFlags, r types.RollbackFlags) error

	Name() string
	Flags() []cli.Flag
	ValidateSettings() error
}

type CDNProvider interface {
	CreateCDN(g types.GlobalFlags, c types.CreateFlags) (cdnDomain string, err error)

	Name() string
	Flags() []cli.Flag
	ValidateSettings() error
}
