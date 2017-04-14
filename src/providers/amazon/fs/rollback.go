package fs

import (
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
)

// Go go the version prefix folder on s3 and copy the html files over to the root as the currently active files
func Rollback(s3Session *s3.S3, domain string, dest string, version string) error {
	prefix := filepath.Join(dest, version) + "/"

	list, err := s3Session.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:  aws.String(domain),
		MaxKeys: aws.Int64(1000),
	})
	if err != nil {
		return err
	}

	if *list.IsTruncated {
		panic(fmt.Sprintf("More than %d HTML files in version, rollback is not supported.  Consider filing a GitHub issue if you need support for this.", list.MaxKeys))
	}
	if len(list.Contents) == 0 {
		log.Printf("A deploy with the provided id (%s) was not found in the specified bucket", version)
		return nil
	}

	wg := sync.WaitGroup{}

	count := 0
	for _, file := range list.Contents {
		wg.Add(1)
		go func(file *s3.Object) {
			defer wg.Done()

			path := *file.Key
			if filepath.Ext(path) != ".html" {
				log.Printf("Skipping non-html file %s", path)
				return
			}

			newPath := filepath.Join(dest, path[len(prefix):])

			log.Printf("Aliasing %s to %s", path, newPath)

			//replace old files with new prefixed files in root
			err := copyFile(s3Session, domain, path, newPath, "text/html", LIMITED)
			if err != nil {
				panic(err)
			}

			count++
		}(file)
	}

	wg.Wait()

	log.Printf("Reverted %d HTML files to version %s", count, version)
	return nil
}
