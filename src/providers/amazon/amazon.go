package amazon

import (
	"errors"
	"io/ioutil"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/s3"

	homedir "github.com/mitchellh/go-homedir"
	ini "github.com/zackbloom/go-ini"

	"github.com/urfave/cli"
)

var Client client

type client struct {
	Key       string `yaml:"key"`
	Secret    string `yaml:"secret"`
	Region    string `yaml:"region"`
	NewUser   bool   `yaml:"new-user"`
	CreateSSL bool   `yaml:"create-custom-ssl"`
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
			Destination: &a.Key,
		},
		cli.StringFlag{
			Name:        "aws-secret",
			Value:       defaultSecret,
			Usage:       "The AWS secret of the provided key",
			Destination: &a.Secret,
		},
		cli.StringFlag{
			Name:        "aws-region",
			Value:       "us-east-1",
			Usage:       "The AWS region the S3 bucket is in",
			Destination: &a.Region,
		},
		cli.BoolFlag{
			Name:        "aws-new-user",
			Usage:       "Create a seperate IAM user for this bucket and distribution",
			Destination: &a.NewUser,
		},
		cli.BoolFlag{
			Name:        "create-custom-ssl",
			Usage:       "Using AWS for a CDN, request a SSL/TLS certificate to support https. Using this command will require email validation to prove you own this domain",
			Destination: &a.CreateSSL,
		},
	}
}

func (a *client) ValidateSettings() error {
	var missingFlags []string
	if a.Key == "" {
		missingFlags = append(missingFlags, "aws-key")
	}
	if a.Secret == "" {
		missingFlags = append(missingFlags, "aws-secret")
	}
	if a.Region == "" {
		missingFlags = append(missingFlags, "aws-region")
	}

	if len(missingFlags) > 0 {
		return errors.New("Missing " + strings.Join(missingFlags, " flag, ") + " flag")
	}

	err := a.SetupAWS()
	if err != nil {
		return err
	}

	return nil
}

var awsSession *session.Session

var s3Session *s3.S3
var iamSession *iam.IAM
var r53Session *route53.Route53
var cfSession *cloudfront.CloudFront

func (a *client) SetupAWS() error {
	creds := credentials.NewStaticCredentials(a.Key, a.Secret, "")
	config := (&aws.Config{
		Region: aws.String(a.Region),
	}).WithCredentials(creds)

	awsSession = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            *config,
	}))

	s3Session = s3.New(awsSession)
	iamSession = iam.New(awsSession)
	r53Session = route53.New(awsSession)
	cfSession = cloudfront.New(awsSession)

	return nil
}