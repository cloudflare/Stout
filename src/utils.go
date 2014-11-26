package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/s3"
	"github.com/imdario/mergo"
	"gopkg.in/yaml.v1"
)

const (
	LIMITED = 60
	FOREVER = 31556926
)

var s3Session *s3.S3

func openS3(key, secret string) *s3.S3 {
	auth := aws.Auth{
		AccessKey: key,
		SecretKey: secret,
	}
	return s3.New(auth, aws.USEast)
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
}

func parseOptions() (o Options) {
	flag.StringVar(&o.Files, "files", "*", "Comma-seperated glob patterns of files to deploy (within root)")
	flag.StringVar(&o.Root, "root", "./", "The local directory to deploy")
	flag.StringVar(&o.Dest, "dest", "./", "The destination directory to write files to in the S3 bucket")
	flag.StringVar(&o.ConfigFile, "config", "", "A yaml file to read configuration from")
	flag.StringVar(&o.Env, "env", "", "The env to read from the config file")
	flag.StringVar(&o.Bucket, "bucket", "", "The bucket to deploy to")
	flag.StringVar(&o.AWSKey, "key", "", "The AWS key to use")
	flag.StringVar(&o.AWSSecret, "secret", "", "The AWS secret of the provided key")

	flag.Parse()

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

func copyFile(bucket *s3.Bucket, from string, to string, contentType string, maxAge int) {
	copyOpts := s3.CopyOptions{
		MetadataDirective: "REPLACE",
		ContentType:       contentType,
		Options: s3.Options{
			CacheControl:    fmt.Sprintf("public, max-age=%d", maxAge),
			ContentEncoding: "gzip",
		},
	}

	_, err := bucket.PutCopy(to, s3.PublicRead, copyOpts, filepath.Join(bucket.Name, from))
	if err != nil {
		panic(err)
	}
}
