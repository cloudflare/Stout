package main

import (
	"fmt"
	"os"

	"github.com/eagerio/Stout/src/actions"
	"github.com/eagerio/Stout/src/config"
	"github.com/eagerio/Stout/src/providers"
	"github.com/eagerio/Stout/src/providers/providermgmt"
	"github.com/eagerio/Stout/src/utils"
	"github.com/urfave/cli"
)

func main() {
	envHolder := providers.EnvHolder{
		GlobalFlags:   &providers.GlobalFlags{},
		CreateFlags:   &providers.CreateFlags{},
		DeployFlags:   &providers.DeployFlags{},
		RollbackFlags: &providers.RollbackFlags{},
	}

	appFlags := []cli.Flag{
		cli.BoolFlag{
			Name:        "debug",
			Usage:       "Display additional debug info",
			Destination: &envHolder.GlobalFlags.Debug,
		},
		cli.StringFlag{
			Name:        "config",
			Value:       "",
			Usage:       "A yaml file to read configuration from",
			Destination: &envHolder.GlobalFlags.Config,
		},
		cli.StringFlag{
			Name:        "env",
			Value:       "",
			Usage:       "The env to read from the config file",
			Destination: &envHolder.GlobalFlags.Env,
		},
		cli.StringFlag{
			Name:        "domain",
			Value:       "",
			Usage:       "The domain to deploy to",
			Destination: &envHolder.GlobalFlags.Domain,
		},
		cli.StringFlag{
			Name:        "dns",
			Value:       "",
			Usage:       "The DNS provider to use",
			Destination: &envHolder.GlobalFlags.DNS,
		},
		cli.StringFlag{
			Name:        "fs",
			Value:       "",
			Usage:       "The file storage provider to use",
			Destination: &envHolder.GlobalFlags.FS,
		},
		cli.StringFlag{
			Name:        "cdn",
			Value:       "",
			Usage:       "The CDN provider to use",
			Destination: &envHolder.GlobalFlags.CDN,
		},
	}

	app := cli.NewApp()
	app.Name = "stout"
	app.Version = "2.0.0"
	app.Usage = "a reliable static website deploy tool"
	app.UsageText = utils.FormattedUsageText()
	app.Commands = []cli.Command{
		{
			Name:  "create",
			Usage: "Configure your CDN, File Storage, and DNS providers for usage with stout.",
			Flags: append(appFlags, providermgmt.CreateCommandFlags()...),
			Action: func(c *cli.Context) (err error) {
				envHolder, err = config.LoadEnvConfig(envHolder)
				if err != nil {
					return err
				}

				return utils.PanicsToErrors(envHolder.GlobalFlags.Debug, func() error {
					return actions.Create(*envHolder.GlobalFlags, *envHolder.CreateFlags)
				})
			},
		},
		{
			Name:  "deploy",
			Usage: "Deploy your static website to your File Storage provider.",
			Flags: append(appFlags, append([]cli.Flag{
				utils.TitleFlag("DEPLOY FLAGS:"),
				cli.StringFlag{
					Name:        "files",
					Value:       "*.html",
					Usage:       "Comma-seperated glob patterns of files to deploy (within root) independently from html referenced js and css files",
					Destination: &envHolder.DeployFlags.Files,
				},
				cli.StringFlag{
					Name:        "root",
					Value:       "./",
					Usage:       "The local directory (prefix) to deploy",
					Destination: &envHolder.DeployFlags.Root,
				},
				cli.StringFlag{
					Name:        "dest",
					Value:       "./",
					Usage:       "The destination directory to write files to in the FS storage location",
					Destination: &envHolder.DeployFlags.Dest,
				},
			}, providermgmt.DeployCommandFlags()...)...),
			Action: func(c *cli.Context) (err error) {
				envHolder, err = config.LoadEnvConfig(envHolder)
				if err != nil {
					return err
				}

				return utils.PanicsToErrors(envHolder.GlobalFlags.Debug, func() error {
					return actions.Deploy(*envHolder.GlobalFlags, *envHolder.DeployFlags)
				})
			},
		},
		{
			Name:  "rollback",
			Usage: "Roll back your website to a specific version.",
			Flags: append(appFlags, append([]cli.Flag{
				utils.TitleFlag("ROLLBACK FLAGS:"),
				cli.StringFlag{
					Name:        "dest",
					Value:       "./",
					Usage:       "The destination directory to write files to in the FS storage location",
					Destination: &envHolder.RollbackFlags.Dest,
				},
				cli.StringFlag{
					Name:        "version",
					Usage:       "The version to rollback to (the version should be the output of the deploy you wish to rollback to)",
					Destination: &envHolder.RollbackFlags.Version,
				},
			}, providermgmt.RollbackCommandFlags()...)...),
			Action: func(c *cli.Context) (err error) {
				envHolder, err = config.LoadEnvConfig(envHolder)
				if err != nil {
					return err
				}

				return utils.PanicsToErrors(envHolder.GlobalFlags.Debug, func() error {
					return actions.Rollback(*envHolder.GlobalFlags, *envHolder.RollbackFlags)
				})
			},
		},
	}

	app.CommandNotFound = func(c *cli.Context, command string) {
		fmt.Fprintf(c.App.Writer, "stout error: %q is not recognized as a valid command.\n", command)
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("stout error: %s\n", err.Error())
	}
}
