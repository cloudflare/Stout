package amazonprovider

import (
	"errors"

	"github.com/urfave/cli"
)

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

func (a *client) Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:        "key",
			Usage:       "The AWS key to use",
			Destination: &a.Info.AWSKey,
		},
		cli.StringFlag{
			Name:        "secret",
			Usage:       "The AWS secret of the provided key",
			Destination: &a.Info.AWSSecret,
		},
		cli.StringFlag{
			Name:        "region",
			Value:       "us-east-1",
			Usage:       "The AWS region the S3 bucket is in",
			Destination: &a.Info.AWSRegion,
		},
	}
}

func (a *client) ValidateSettings(c cli.Context) error {
	if c.String("key") == "" {
		return errors.New("Missing AWS key flag")
	}
	if c.String("secret") == "" {
		return errors.New("Missing AWS secret flag")
	}
	if c.String("region") == "" {
		return errors.New("Missing AWS region flag")
	}
	return nil
}
