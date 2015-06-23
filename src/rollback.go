package main

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/zackbloom/goamz/s3"
)

func Rollback(options Options, version string) {
	if s3Session == nil {
		s3Session = openS3(options.AWSKey, options.AWSSecret, options.AWSRegion)
	}

	bucket := s3Session.Bucket(options.Bucket)

	// List files with the correct prefix in bucket
	// Remove their prefix with a copy.

	prefix := filepath.Join(options.Dest, version) + "/"

	list, err := bucket.List(prefix, "", "", 1000)
	panicIf(err)

	if list.IsTruncated {
		panic(fmt.Sprintf("More than %d HTML files in version, rollback is not supported.  Consider filing a GitHub issue if you need support for this.", list.MaxKeys))
	}
	if len(list.Contents) == 0 {
		log.Printf("A deploy with the provided id (%s) was not found in the specified bucket", version)
		return
	}

	wg := sync.WaitGroup{}

	count := 0
	for _, file := range list.Contents {
		wg.Add(1)
		go func(file s3.Key) {
			defer wg.Done()

			path := file.Key
			if filepath.Ext(path) != ".html" {
				log.Printf("Skipping non-html file %s", path)
				return
			}

			newPath := filepath.Join(options.Dest, path[len(prefix):])

			log.Printf("Aliasing %s to %s", path, newPath)

			copyFile(bucket, path, newPath, "text/html", LIMITED)

			count++
		}(file)
	}

	wg.Wait()

	log.Printf("Reverted %d HTML files to version %s", count, version)
}

func rollbackCmd() {
	options, set := parseOptions()
	version := set.Arg(0)

	loadConfigFile(&options)
	addAWSConfig(&options)

	if options.Bucket == "" {
		panic("You must specify a bucket")
	}
	if options.AWSKey == "" || options.AWSSecret == "" {
		panic("You must specify your AWS credentials")
	}
	if version == "" {
		panic("You must specify a version to rollback to")
	}

	Rollback(options, version)
}
