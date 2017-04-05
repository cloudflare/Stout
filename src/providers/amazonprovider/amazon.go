package amazonprovider

import (
	"errors"
	"io/ioutil"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/urfave/cli"
	ini "github.com/zackbloom/go-ini"
)

var Client client

type client struct {
	AWSKey     string `yaml:"key"`
	AWSSecret  string `yaml:"secret"`
	AWSRegion  string `yaml:"region"`
	AWSNewUser bool   `yaml:"newuser"`
}

func (a *client) Name() string {
	return "amazon"
}

// Struct to represent the AWS config
type AWSConfig struct {
	Default struct {
		AccessKey string `ini:"aws_access_key_id"`
		SecretKey string `ini:"aws_secret_access_key"`
	} `ini:"[default]"`
}

// load the aws config from ~/.aws/
func loadAWSConfig() (access string, secret string) {
	cfg := AWSConfig{}

	//TODO: support windows loation for aws credentials
	for _, file := range []string{"~/.aws/config", "~/.aws/credentials"} {
		path, err := homedir.Expand(file)
		if err != nil {
			continue
		}

		content, err := ioutil.ReadFile(path)
		if err != nil {
			continue
		}

		ini.Unmarshal(content, &cfg)

		if cfg.Default.AccessKey != "" {
			break
		}
	}

	return cfg.Default.AccessKey, cfg.Default.SecretKey
}

func (a *client) Flags() []cli.Flag {
	defaultKey, defaultSecret := loadAWSConfig()

	return []cli.Flag{
		cli.StringFlag{
			Name:        "aws-key",
			Value:       defaultKey,
			Usage:       "The AWS key to use",
			Destination: &a.AWSKey,
		},
		cli.StringFlag{
			Name:        "aws-secret",
			Value:       defaultSecret,
			Usage:       "The AWS secret of the provided key",
			Destination: &a.AWSSecret,
		},
		cli.StringFlag{
			Name:        "aws-region",
			Value:       "us-east-1",
			Usage:       "The AWS region the S3 bucket is in",
			Destination: &a.AWSRegion,
		},
		cli.BoolFlag{
			Name:        "aws-new-user",
			Usage:       "Create a seperate IAM user for this bucket and distribution",
			Destination: &a.AWSNewUser,
		},
	}
}

func (a *client) ValidateSettings() error {
	if a.AWSKey == "" {
		return errors.New("Missing aws-key flag")
	}
	if a.AWSSecret == "" {
		return errors.New("Missing aws-secret flag")
	}
	if a.AWSRegion == "" {
		return errors.New("Missing aws-region flag")
	}

	err := setupAWS(a.AWSKey, a.AWSSecret, a.AWSRegion)
	if err != nil {
		return err
	}

	return nil
}
