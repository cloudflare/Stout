package main

import (
	"fmt"
	"os"

	"github.com/eagerio/Stout/src/actions"
	"github.com/eagerio/Stout/src/providers"
	"github.com/urfave/cli"
)

/*
* Prints a brief description of the usage of the tool
 */
func printUsageDescription() {
	fmt.Println(
		`Stout Static Deploy Tool
Supports three commands: create, deploy and rollback.

Example Usage:
 To create a site which will be hosted at my.awesome.website:
   stout create --domain my.awesome.website --key AWS_KEY --secret AWS_SECRET

 To deploy the current folder to the root of the my.awesome.website site:
  stout deploy --domain my.awesome.website --key AWS_KEY --secret AWS_SECRET

 To rollback to a specific deploy:
  stout rollback --domain my.awesome.website --key AWS_KEY --secret AWS_SECRET c4a22bf94de1

 See the README for more configuration information.
 run stout help for all options"

`)
}

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

// set.StringVar(&o.DNSProvider, "dns", "", "The DNS provider to use")
// set.StringVar(&o.FSProvider, "file-storage", "", "The file storage provider to use")
// set.StringVar(&o.CDNProvider, "cdn", "", "The CDN provider to use")

// func checkForRequiredOptions(options Options) {
// 	if options.Domain == "" {
// 		panic("You must specify a domain")
// 	}
//
// 	if !validProviders[options.DNSProvider] || !validProviders[options.FSProvider] || !validProviders[options.CDNProvider] {
// 		panic("You must specify a valid DNS, file storage, and CDN provider")
// 	}
//
// 	if options.AWSKey == "" || options.AWSSecret == "" {
// 		panic("You must specify your AWS credentials")
// 	}
// }

func main() {
	app := cli.NewApp()
	app.Name = "stout"
	app.Version = "2.0.0"
	app.Usage = "a reliable static website deploy tool"
	app.UsageText = "stout [global options] <command> [command options], or" + "\n" +
		"   stout help <command>, to learn more about a subcommand"

	// addAWSConfig(&options)
	// checkForRequiredOptions(options)

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config",
			Value: "",
			Usage: "A yaml file to read configuration from",
		},
		cli.StringFlag{
			Name:  "env",
			Value: "",
			Usage: "The env to read from the config file",
		},
		cli.StringFlag{
			Name:  "domain",
			Value: "",
			Usage: "The domain to deploy to",
		},
		cli.StringFlag{
			Name:  "dns",
			Value: "",
			Usage: "The DNS provider to use",
		},
		cli.StringFlag{
			Name:  "fs",
			Value: "",
			Usage: "The file storage provider to use",
		},
		cli.StringFlag{
			Name:  "cdn",
			Value: "",
			Usage: "The CDN provider to use",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "create",
			Usage: "Configure your CDN, File Storage, and DNS providers for usage with stout.",
			Flags: append([]cli.Flag{
				cli.BoolFlag{
					Name:  "create-ssl",
					Usage: "Request a SSL/TLS certificate to support https. Using this command will require email validation to prove you own this domain",
				},
				cli.BoolFlag{
					Name:  "no-ssl",
					Usage: "Do not set up SSL/TLS certificates",
				},
			}, providers.CommandFlags(true, true, true)...),
			Action: func(c *cli.Context) (err error) {
				defer func() {
					if r := recover(); r != nil {
						var ok bool
						err, ok = r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
					}
				}()

				return actions.Create(c)
			},
		},
		{
			Name:  "deploy",
			Usage: "Deploy your static website to your File Storage provider.",
			Action: func(c *cli.Context) error {
				deploySubcommand := cli.NewApp()
				deploySubcommand.Flags = []cli.Flag{
					cli.StringFlag{
						Name:  "files",
						Value: "*",
						Usage: "Comma-seperated glob patterns of files to deploy (within root) independently from html referenced js and css files",
					},
					cli.StringFlag{
						Name:  "root",
						Value: "./",
						Usage: "The local directory (prefix) to deploy",
					},
					cli.StringFlag{
						Name:  "dest",
						Value: "./",
						Usage: "The destination directory to write files to in the S3 bucket",
					},
				}

				deploySubcommand.Action = func(c *cli.Context) error {
					// Deploy()
					return nil
				}

				return deploySubcommand.RunAsSubcommand(c)
			},
		},
		{
			Name:  "rollback",
			Usage: "Roll back your website to a specific version.",
			Action: func(c *cli.Context) error {
				version := c.Args().First()
				if version == "" {
					panic("You must specify a version to rollback to")
				}

				// Rollback()

				return nil
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
