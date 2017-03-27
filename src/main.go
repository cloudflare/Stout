package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/eagerio/Stout/src/actions"
	"github.com/eagerio/Stout/src/providers"
	"github.com/eagerio/Stout/src/types"
	"github.com/eagerio/Stout/src/utils"
	"github.com/urfave/cli"
)

type Options struct {
	Files       string `yaml:"files"`
	Root        string `yaml:"root"`
	Dest        string `yaml:"dest"`
	ConfigFile  string `yaml:"-"`
	Env         string `yaml:"-"`
	Domain      string `yaml:"domain"`
	NoUser      bool   `yaml:"-"`
	NoSSL       bool   `yaml:"-"`
	CreateSSL   bool   `yaml:"-"`
	DNSProvider string `yaml:"dns"`
	FSProvider  string `yaml:"file-storage"`
	CDNProvider string `yaml:"cdn"`
	AWSKey      string `yaml:"key"`
	AWSSecret   string `yaml:"secret"`
	AWSRegion   string `yaml:"region"`
}

func formattedUsageText() string {
	text := (`
stout [global options] <command> [command options], or
stout help <command>, to learn more about a subcommand

Example Usage:

To create a site which will be hosted at my.awesome.website:
  stout --fs amazon --cdn amazon --dns amazon create --domain my.awesome.website --key AWS_KEY --secret AWS_SECRET

To deploy the current folder to the root of the my.awesome.website site:
  stout --fs amazon deploy --domain my.awesome.website --key AWS_KEY --secret AWS_SECRET

To rollback to a specific deploy:
  stout --fs amazon rollback --domain my.awesome.website --key AWS_KEY --secret AWS_SECRET c4a22bf94de1
 `)

	textArray := strings.Split(text, "\n")
	formattedText := strings.Join(textArray[1:], "\n   ")

	return formattedText
}

func main() {
	var globalFlagHolder types.GlobalFlags
	var createFlagHolder types.CreateFlags
	var deployFlagHolder types.DeployFlags
	var rollbackFlagHolder types.RollbackFlags

	app := cli.NewApp()
	app.Name = "stout"
	app.Version = "2.0.0"
	app.Usage = "a reliable static website deploy tool"
	app.UsageText = formattedUsageText()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "config",
			Value:       "",
			Usage:       "A yaml file to read configuration from",
			Destination: &globalFlagHolder.Config,
		},
		cli.StringFlag{
			Name:        "env",
			Value:       "",
			Usage:       "The env to read from the config file",
			Destination: &globalFlagHolder.Env,
		},
		cli.StringFlag{
			Name:        "domain",
			Value:       "",
			Usage:       "The domain to deploy to",
			Destination: &globalFlagHolder.Domain,
		},
		cli.StringFlag{
			Name:        "dns",
			Value:       "",
			Usage:       "The DNS provider to use",
			Destination: &globalFlagHolder.DNS,
		},
		cli.StringFlag{
			Name:        "fs",
			Value:       "",
			Usage:       "The file storage provider to use",
			Destination: &globalFlagHolder.FS,
		},
		cli.StringFlag{
			Name:        "cdn",
			Value:       "",
			Usage:       "The CDN provider to use",
			Destination: &globalFlagHolder.CDN,
		},
	}
	app.Commands = []cli.Command{
		{
			Name:  "create",
			Usage: "Configure your CDN, File Storage, and DNS providers for usage with stout.",
			Flags: append([]cli.Flag{
				cli.BoolFlag{
					Name:        "create-ssl",
					Usage:       "Request a SSL/TLS certificate to support https. Using this command will require email validation to prove you own this domain",
					Destination: &createFlagHolder.CreateSSL,
				},
				cli.BoolFlag{
					Name:        "no-ssl",
					Usage:       "Do not set up SSL/TLS certificates",
					Destination: &createFlagHolder.NoSSL,
				},
			}, providers.CreateCommandFlags()...),
			Action: func(c *cli.Context) (err error) {
				return utils.PanicsToErrors(func() error {
					return actions.Create(globalFlagHolder, createFlagHolder)
				})
			},
		},
		{
			Name:  "deploy",
			Usage: "Deploy your static website to your File Storage provider.",
			Flags: append([]cli.Flag{
				cli.StringFlag{
					Name:        "files",
					Value:       "*",
					Usage:       "Comma-seperated glob patterns of files to deploy (within root) independently from html referenced js and css files",
					Destination: &deployFlagHolder.Files,
				},
				cli.StringFlag{
					Name:        "root",
					Value:       "./",
					Usage:       "The local directory (prefix) to deploy",
					Destination: &deployFlagHolder.Root,
				},
				cli.StringFlag{
					Name:        "dest",
					Value:       "./",
					Usage:       "The destination directory to write files to in the FS storage location",
					Destination: &deployFlagHolder.Dest,
				},
			}, providers.DeployCommandFlags()...),
			Action: func(c *cli.Context) (err error) {
				return utils.PanicsToErrors(func() error {
					return actions.Deploy(globalFlagHolder, deployFlagHolder)
				})
			},
		},
		{
			Name:  "rollback",
			Usage: "Roll back your website to a specific version.",
			Flags: append([]cli.Flag{
				cli.StringFlag{
					Name:        "dest",
					Value:       "./",
					Usage:       "The destination directory to write files to in the FS storage location",
					Destination: &rollbackFlagHolder.Dest,
				},
				cli.StringFlag{
					Name:        "version",
					Usage:       "The version to rollback to (the version should be the output of the deploy you wish to rollback to)",
					Destination: &rollbackFlagHolder.Version,
				},
			}, providers.RollbackCommandFlags()...),
			Action: func(c *cli.Context) (err error) {
				return utils.PanicsToErrors(func() error {
					return actions.Rollback(globalFlagHolder, rollbackFlagHolder)
				})
			},
		},
	}

	app.CommandNotFound = func(c *cli.Context, command string) {
		fmt.Fprintf(c.App.Writer, "stout error: %q is not recognized as a valid command.\n", command)
	}
	// app.Before = altsrc.InitInputSourceWithContext(flags, altsrc.NewYamlSourceFromFlagFunc("config"))

	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("stout error: %s\n", err.Error())
	}
}
