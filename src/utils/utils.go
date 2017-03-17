package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/imdario/mergo"
	"github.com/mitchellh/go-homedir"
	"github.com/zackbloom/go-ini"
	"github.com/zackbloom/goamz/s3"
	"gopkg.in/yaml.v1"
)

/*
* Catch errors and panic if there is an error
 */
func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}

/*
* Catch errors and panic if there is an error
 */
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

type ConfigFile map[string]Options

/*
* Load config file: this is called from the cli to populate options from the config file
 */
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

	panicIf(mergo.MergeWithOverwrite(o, defCfg))
	panicIf(mergo.MergeWithOverwrite(o, envCfg))
}

/*
* Load aws config to the options
 */
func addAWSConfig(o *Options) {
	if o.AWSKey == "" && o.AWSSecret == "" {
		o.AWSKey, o.AWSSecret = loadAWSConfig()
	}
}

/*
* Struct to represent the AWS config
 */
type AWSConfig struct {
	Default struct {
		AccessKey string `ini:"aws_access_key_id"`
		SecretKey string `ini:"aws_secret_access_key"`
	} `ini:"[default]"`
}

/*
* load the aws config from ~/.aws/
 */
func loadAWSConfig() (access string, secret string) {
	cfg := AWSConfig{}

	//TODO(renandincer): support windows loation for aws credentials
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

/*
* Copy file in s3
 */
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

/*
* Merge files using forward slashes and not the system path seperator if that is different
* Useful since windows has backslash path separators instead of forward slash which is hard to use with S3
 */
func joinPath(parts ...string) string {
	// Like filepath.Join, but always uses '/'
	out := filepath.Join(parts...)

	if os.PathSeparator != '/' {
		out = strings.Replace(out, string(os.PathSeparator), "/", -1)
	}

	return out
}
