package google

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"github.com/urfave/cli"
)

var ctx context.Context
var gclient *storage.Client

var Client client

type client struct {
	Keyfile   string `yaml:"keyfile"`
	ProjectID string `yaml:"project-id"`
	Location  string `yaml:"location"`
}

func (c *client) Name() string {
	return "google"
}

func (c *client) Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:        "google-keyfile",
			Usage:       "Location of the service account json keyfile containing auth data",
			Destination: &c.Keyfile,
		},
		cli.StringFlag{
			Name:        "google-project-id",
			Usage:       "Project ID (not name) of the project to create the bucket inside of",
			Destination: &c.ProjectID,
		},
		cli.StringFlag{
			Name:        "google-bucket-location",
			Usage:       "Location of the bucket",
			Value:       "US",
			Destination: &c.Location,
		},
	}
}

func (c *client) ValidateSettings() error {
	var missingFlags []string
	if c.Keyfile == "" {
		missingFlags = append(missingFlags, "google-keyfile")
	}
	if c.ProjectID == "" {
		missingFlags = append(missingFlags, "google-project-id")
	}

	if len(missingFlags) > 0 {
		return errors.New("Missing " + strings.Join(missingFlags, " flag, ") + " flag")
	}

	err := c.setup()
	if err != nil {
		return err
	}

	return nil
}

func (c *client) setup() error {
	ctx = context.Background()

	var err error
	gclient, err = storage.NewClient(ctx)

	// using the following line of code, will error out, commenting out keyfile based auth until this is resolved:
	gclient, err = storage.NewClient(ctx, option.WithServiceAccountFile(c.Keyfile))

	return err
}
