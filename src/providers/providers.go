package providers

import (
	"github.com/urfave/cli"
)

type ProviderClient interface {
	Name() string
	Flags() []cli.Flag
	ValidateSettings() error
}

type DNSProvider interface {
	CreateDNS(g GlobalFlags, c CreateFlags, cdnDomain string) error

	Name() string
	Flags() []cli.Flag
	ValidateSettings() error
}

type FSProvider interface {
	CreateFS(g GlobalFlags, c CreateFlags) error
	DeployFS(g GlobalFlags, d DeployFlags) error
	RollbackFS(g GlobalFlags, r RollbackFlags) error

	Name() string
	Flags() []cli.Flag
	ValidateSettings() error
}

type CDNProvider interface {
	CreateCDN(g GlobalFlags, c CreateFlags) (cdnDomain string, err error)

	Name() string
	Flags() []cli.Flag
	ValidateSettings() error
}
