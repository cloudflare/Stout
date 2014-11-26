package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
)

func Rollback(options Options, version string) {
	if s3Session == nil {
		s3Session = openS3(options.AWSKey, options.AWSSecret)
	}

	bucket := s3Session.Bucket(options.Bucket)

	// List files with the correct prefix in bucket
	// Remove their prefix with a copy.

	list, err := bucket.List(version+"/", "", "", 1000)
	panicIf(err)

	if list.IsTruncated {
		panic(fmt.Sprintf("More than %d HTML files in version, rollback is not supported.  Consider filing a GitHub issue if you need support for this.", list.MaxKeys))
	}
	if len(list.Contents) == 0 {
		log.Printf("A deploy with the provided id (%s) was not found in the specified bucket", version)
		return
	}

	count := 0
	for _, file := range list.Contents {
		path := file.Key
		if filepath.Ext(path) != ".html" {
			log.Printf("Skipping non-html file %s", path)
			continue
		}

		newPath := path[len(version)+1:]

		log.Printf("Replacing %s with %s", path, newPath)

		copyFile(bucket, filepath.Join(options.Bucket, path), newPath, "text/html", LIMITED)

		count++
	}

	log.Printf("Reverted %d HTML files to version %s", count, version)
}

func rollbackCmd() {
	options := parseOptions()
	version := flag.Arg(1)

	loadConfigFile(&options)

	if options.Bucket == "" {
		panic("You must specify a bucket")
	}
	if options.AWSKey == "" || options.AWSSecret == "" {
		panic("You must specify your AWS credentials")
	}
	if version == "" {
		panic("You must specify a version to rollback to")
	}

	s3Session = openS3(options.AWSKey, options.AWSSecret)

	Rollback(options, version)
}
