package main

import (
	"flag"
	"fmt"
	"os"
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
   stout create --bucket my.awesome.website --key AWS_KEY --secret AWS_SECRET

 To deploy the current folder to the root of the my.awesome.website site:
  stout deploy --bucket my.awesome.website --key AWS_KEY --secret AWS_SECRET

 To rollback to a specific deploy:
  stout rollback --bucket my.awesome.website --key AWS_KEY --secret AWS_SECRET c4a22bf94de1

 See the README for more configuration information.
 run stout help for all options"

`)
}

/*
* Options that are supplied either from the cli flags or a deploy.yaml file
 */
type Options struct {
	Files      string `yaml:"files"`
	Root       string `yaml:"root"`
	Dest       string `yaml:"dest"`
	ConfigFile string `yaml:"-"`
	Env        string `yaml:"-"`
	Bucket     string `yaml:"bucket"`
	AWSKey     string `yaml:"key"`
	AWSSecret  string `yaml:"secret"`
	AWSRegion  string `yaml:"region"`
	NoUser     bool   `yaml:"-"`
	NoSSL      bool   `yaml:"-"`
	CreateSSL  bool   `yaml:"-"`
}

/*
* Parses the command line flags into options
 */
func parseOptions() (o Options, set *flag.FlagSet) {
	set = flag.NewFlagSet("flags", flag.ExitOnError)

	set.StringVar(&o.Files, "files", "*", "Comma-seperated glob patterns of files to deploy (within root)")
	set.StringVar(&o.Root, "root", "./", "The local directory to deploy")
	set.StringVar(&o.Dest, "dest", "./", "The destination directory to write files to in the S3 bucket")
	set.StringVar(&o.ConfigFile, "config", "", "A yaml file to read configuration from")
	set.StringVar(&o.Env, "env", "", "The env to read from the config file")
	set.StringVar(&o.Bucket, "bucket", "", "The bucket to deploy to")
	set.StringVar(&o.AWSKey, "key", "", "The AWS key to use")
	set.StringVar(&o.AWSSecret, "secret", "", "The AWS secret of the provided key")
	set.StringVar(&o.AWSRegion, "region", "us-east-1", "The AWS region the S3 bucket is in")
	set.BoolVar(&o.NoUser, "no-user", false, "Should a seperate IAM user be created for this bucket and distribution?")
	set.BoolVar(&o.CreateSSL, "create-ssl", false, "Request a SSL/TLS certificate to support https")
	set.BoolVar(&o.NoSSL, "no-ssl", false, "Do not set up SSL/TLS certificates")

	// if there us anything to parse
	if len(os.Args) > 1 {
		set.Parse(os.Args[2:])
	}

	return
}

/*
* Checks if the bucket is specified and aws credentials supplied
 */
func checkForBucketAndKeys(options Options) {
	if options.Bucket == "" {
		panic("You must specify a bucket")
	}

	if options.AWSKey == "" || options.AWSSecret == "" {
		panic("You must specify your AWS credentials")
	}
}

/*
* Entrypoint
* 1. Parse flags to determine command
* 2. Parse flags into options and run that command
* If command is not found, return an error
 */
func main() {
	flag.Parse()
	command := flag.Arg(0)

	options, flagSet := parseOptions()

	loadConfigFile(&options)
	addAWSConfig(&options)

	switch command {
	case "help":
		printUsageDescription()
		fmt.Println("Available options:")
		flagSet.PrintDefaults()
	case "deploy":
		checkForBucketAndKeys(options)
		Deploy(options)
	case "rollback":
		version := flagSet.Arg(0)
		checkForBucketAndKeys(options)
		if version == "" {
			panic("You must specify a version to rollback to")
		}
		Rollback(options, version)
	case "create":
		checkForBucketAndKeys(options)
		Create(options)
	default:
		fmt.Println("Command not understood")
		fmt.Println("run stout help for all options")
	}
}
