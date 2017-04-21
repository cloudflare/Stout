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

type FSProvider interface {
	CreateFS(g types.GlobalFlags, c types.CreateFlags) (fsDomain string, err error)

	FSProviderFuncs(g types.GlobalFlags) (types.FSProviderFunctions, error)

	Name() string
	Flags() []cli.Flag
	ValidateSettings() error
}

type CDNProvider interface {
	CreateCDN(g types.GlobalFlags, c types.CreateFlags, fsDomain string) (cdnDomain string, err error)

	Name() string
	Flags() []cli.Flag
	ValidateSettings() error
}

type DNSProvider interface {
	CreateDNS(g types.GlobalFlags, c types.CreateFlags, cdnDomain string) error

	AddVerificationRecord(g types.GlobalFlags, c types.CreateFlags, recordType types.DNSRecordType, name string, value string) error

	Name() string
	Flags() []cli.Flag
	ValidateSettings() error
}
