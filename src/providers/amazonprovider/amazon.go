package amazonprovider

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/urfave/cli"
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

func (a *client) Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:        "aws-key",
			Usage:       "The AWS key to use",
			Destination: &a.AWSKey,
		},
		cli.StringFlag{
			Name:        "aws-secret",
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

	err := checkForAWS()
	if err != nil {
		return err
	}

	//official sdk connection
	if awsSession == nil {
		awsSession = session.New(&aws.Config{
			Region:      aws.String(a.AWSRegion),
			Credentials: credentials.NewStaticCredentials(a.AWSKey, a.AWSSecret, ""),
		})
	}

	// open all services sessions
	if s3Session == nil {
		s3Session = openS3(a.AWSKey, a.AWSSecret, a.AWSRegion)
	}
	if iamSession == nil {
		iamSession = openIAM(a.AWSKey, a.AWSSecret, a.AWSRegion)
	}
	if r53Session == nil {
		r53Session = openRoute53(a.AWSKey, a.AWSSecret)
	}
	if cfSession == nil {
		cfSession = openCloudFront(a.AWSKey, a.AWSSecret)
	}

	return nil
}
