package cloudflare

import (
	"errors"
	"strings"

	cloudflare "github.com/cloudflare/cloudflare-go"
	"github.com/urfave/cli"
)

var api *cloudflare.API

var Client client

type client struct {
	Email string `yaml:"email"`
	Key   string `yaml:"key"`
}

func (c *client) Name() string {
	return "cloudflare"
}

func (c *client) Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:        "cf-email",
			Usage:       "Cloudflare account email",
			Destination: &c.Email,
		},
		cli.StringFlag{
			Name:        "cf-key",
			Usage:       "Cloudflare account API key",
			Destination: &c.Key,
		},
	}
}

func (c *client) ValidateSettings() error {
	var missingFlags []string
	if c.Email == "" {
		missingFlags = append(missingFlags, "cf-email")
	}
	if c.Key == "" {
		missingFlags = append(missingFlags, "cf-key")
	}

	if len(missingFlags) > 0 {
		return errors.New("Missing " + strings.Join(missingFlags, " flag, ") + " flag")
	}

	err := c.setupCloudflare()
	if err != nil {
		return err
	}

	return nil
}

func (c *client) setupCloudflare() error {
	var err error
	api, err = cloudflare.New(c.Key, c.Email)
	return err
}
