package providers

import "github.com/eagerio/Stout/src/providers/amazonprovider"

type ProviderHolder struct {
	DNSProvider
	FSProvider
	CDNProvider
}

var ProviderList = map[string]ProviderClient{
	amazonprovider.Client.Name(): &amazonprovider.Client,
}

type ProviderClient interface {
	Name() string
	SetFlags()
	ValidateSettings() error
}

type DNSProvider interface {
	CreateDNS() error
}

type FSProvider interface {
	CreateFS() error
	DeployFS() error
	RollbackFS() error
}

type CDNProvider interface {
	CreateCDN() error
}
