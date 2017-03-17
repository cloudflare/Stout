package amazonprovider

var Client client

type client struct {
	Info amazonInfo
}

type amazonInfo struct {
	AWSKey    string `yaml:"key"`
	AWSSecret string `yaml:"secret"`
	AWSRegion string `yaml:"region"`
}

func (a *client) Name() string {
	return "amazon"
}

func (a *client) SetFlags() {
	// flagHelper.ProviderSet(a, &a.Info.AWSKey, "key", "", "The AWS key to use")
	// flagHelper.ProviderSet(a, &a.Info.AWSSecret, "secret", "", "The AWS secret of the provided key")
	// flagHelper.ProviderSet(a, &a.Info.AWSRegion, "region", "us-east-1", "The AWS region the S3 bucket is in")
}

func (a *client) ValidateSettings() error {
	return nil
}
