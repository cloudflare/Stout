package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"

	"github.com/imdario/mergo"
	"github.com/mitchellh/go-homedir"
	"github.com/zackbloom/go-ini"
	"github.com/zackbloom/goamz/aws"
	"github.com/zackbloom/goamz/cloudfront"
	"github.com/zackbloom/goamz/iam"
	"github.com/zackbloom/goamz/route53"
	"github.com/zackbloom/goamz/s3"
	"gopkg.in/yaml.v1"
)

const (
	LIMITED = 60
	FOREVER = 31556926
)

var s3Session *s3.S3
var iamSession *iam.IAM
var r53Session *route53.Route53
var cfSession *cloudfront.CloudFront

func getRegion(region string) aws.Region {
	regionS, ok := aws.Regions[region]
	if !ok {
		panic("Region not found")
	}
	return regionS
}

func openS3(key, secret, region string) *s3.S3 {
	regionS := getRegion(region)

	auth := aws.Auth{
		AccessKey: key,
		SecretKey: secret,
	}
	return s3.New(auth, regionS)
}

func openIAM(key, secret, region string) *iam.IAM {
	regionS := getRegion(region)

	auth := aws.Auth{
		AccessKey: key,
		SecretKey: secret,
	}
	return iam.New(auth, regionS)
}

func openCloudFront(key, secret string) *cloudfront.CloudFront {
	auth := aws.Auth{
		AccessKey: key,
		SecretKey: secret,
	}
	return cloudfront.NewCloudFront(auth)
}

func openRoute53(key, secret string) *route53.Route53 {
	auth := aws.Auth{
		AccessKey: key,
		SecretKey: secret,
	}

	r53, _ := route53.NewRoute53(auth)
	return r53
}

func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}
func must(val interface{}, err error) interface{} {
	if err != nil {
		panic(err)
	}

	return val
}
func mustString(val string, err error) string {
	panicIf(err)
	return val
}
func mustInt(val int, err error) int {
	panicIf(err)
	return val
}

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
}

func parseOptions() (o Options, set *flag.FlagSet) {
	set = flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	//TODO: Set set.Usage

	set.StringVar(&o.Files, "files", "*", "Comma-seperated glob patterns of files to deploy (within root)")
	set.StringVar(&o.Root, "root", "./", "The local directory to deploy")
	set.StringVar(&o.Dest, "dest", "./", "The destination directory to write files to in the S3 bucket")
	set.StringVar(&o.ConfigFile, "config", "", "A yaml file to read configuration from")
	set.StringVar(&o.Env, "env", "", "The env to read from the config file")
	set.StringVar(&o.Bucket, "bucket", "", "The bucket to deploy to")
	set.StringVar(&o.AWSKey, "key", "", "The AWS key to use")
	set.StringVar(&o.AWSSecret, "secret", "", "The AWS secret of the provided key")
	set.StringVar(&o.AWSRegion, "region", "us-east-1", "The AWS region the S3 bucket is in")
	set.BoolVar(&o.NoUser, "no-user", false, "When creating, should we make a user account?")

	set.Parse(os.Args[2:])

	return
}

type ConfigFile map[string]Options

func loadConfigFile(o *Options) {
	isDefault := false
	configPath := o.ConfigFile
	if o.ConfigFile == "" {
		isDefault = true
		configPath = "./deploy.yaml"
	}

	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) && isDefault {
			return
		}

		panic(err)
	}

	var file ConfigFile
	err = yaml.Unmarshal(data, &file)
	panicIf(err)

	var envCfg Options
	if o.Env != "" {
		var ok bool
		envCfg, ok = file[o.Env]
		if !ok {
			panic("Config for specified env not found")
		}
	}

	defCfg, _ := file["default"]

	panicIf(mergo.Merge(o, defCfg))
	panicIf(mergo.Merge(o, envCfg))
}

func addAWSConfig(o *Options) {
	if o.AWSKey == "" && o.AWSSecret == "" {
		o.AWSKey, o.AWSSecret = loadAWSConfig()
	}
}

type AWSConfig struct {
	Default struct {
		AccessKey string `ini:"aws_access_key_id"`
		SecretKey string `ini:"aws_secret_access_key"`
	} `ini:"[default]"`
}

func loadAWSConfig() (access string, secret string) {
	cfg := AWSConfig{}

	path, err := homedir.Expand("~/.aws/config")
	if err != nil {
		return
	}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return
	}

	ini.Unmarshal(content, &cfg)

	return cfg.Default.AccessKey, cfg.Default.SecretKey
}

func copyFile(bucket *s3.Bucket, from string, to string, contentType string, maxAge int) {
	copyOpts := s3.CopyOptions{
		MetadataDirective: "REPLACE",
		ContentType:       contentType,
		Options: s3.Options{
			CacheControl:    fmt.Sprintf("public, max-age=%d", maxAge),
			ContentEncoding: "gzip",
		},
	}

	_, err := bucket.PutCopy(to, s3.PublicRead, copyOpts, joinPath(bucket.Name, from))
	if err != nil {
		panic(err)
	}
}

var pathRe = regexp.MustCompile("/{2,}")

func joinPath(parts ...string) string {
	// Like filepath.Join, but always uses '/'
	return pathRe.ReplaceAllLiteralString(strings.Join(parts, "/"), "/")
}
