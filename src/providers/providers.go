package providers

import "github.com/urfave/cli"

type ProviderClient interface {
	Name() string
	Flags() []cli.Flag
	ValidateSettings(cli.Context) error
}

type DNSProvider interface {
	CreateDNS(cli.Context) error

	Name() string
	Flags() []cli.Flag
	ValidateSettings(cli.Context) error
}

type FSProvider interface {
	CreateFS(cli.Context) error
	DeployFS(cli.Context) error
	RollbackFS(cli.Context) error

	Name(cli.Context) string
	Flags() []cli.Flag
	ValidateSettings(cli.Context) error
}

type CDNProvider interface {
	CreateCDN(cli.Context) error

	Name() string
	Flags() []cli.Flag
	ValidateSettings(cli.Context) error
}
