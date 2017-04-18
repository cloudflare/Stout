package github

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
	"github.com/urfave/cli"
)

var ghclient *github.Client
var ctx context.Context

var Client client

type client struct {
	Username string `yaml:"username"`
	Token    string `yaml:"token"`
	Reponame string `yaml:"repo-name"`
}

func (c *client) Name() string {
	return "github"
}

func (c *client) Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:        "gh-username",
			Usage:       "Github username",
			Destination: &c.Username,
		},
		cli.StringFlag{
			Name:        "gh-token",
			Usage:       "Github personal access token",
			Destination: &c.Token,
		},
		cli.StringFlag{
			Name:        "gh-repo-name",
			Usage:       "Github repo name to use",
			Destination: &c.Reponame,
		},
	}
}

func (c *client) ValidateSettings() error {
	var missingFlags []string
	if c.Username == "" {
		missingFlags = append(missingFlags, "gh-username")
	}
	if c.Token == "" {
		missingFlags = append(missingFlags, "gh-token")
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
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: c.Token},
	)
	tc := oauth2.NewClient(ctx, ts)

	ghclient = github.NewClient(tc)
	return nil
}
