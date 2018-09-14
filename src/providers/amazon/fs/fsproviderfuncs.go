package fs

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cloudflare/stout/src/types"
)

func FSProviderFuncs(s3Session *s3.S3, domain string) (types.FSProviderFunctions, error) {
	return types.FSProviderFunctions{
		UploadFile: func(f types.UploadFileHolder) error {
			_, err := s3Session.PutObject(&s3.PutObjectInput{
				ACL:             aws.String(s3.BucketCannedACLPublicRead),
				Body:            bytes.NewReader(f.Body),
				Bucket:          aws.String(domain),
				ContentLength:   aws.Int64(int64(len(f.Body))),
				ContentType:     aws.String(f.ContentType + "; charset=utf-8"),
				ContentEncoding: aws.String(f.ContentEncoding),
				Key:             aws.String(f.Dest),
				CacheControl:    aws.String(fmt.Sprintf("public, max-age=%d", f.CacheSeconds)),
			})

			return err
		},

		CopyFile: func(f types.CopyFileHolder) error {
			_, err := s3Session.CopyObject(&s3.CopyObjectInput{
				Bucket:            aws.String(domain),
				MetadataDirective: aws.String("REPLACE"),
				Key:               aws.String(f.Dest),
				ContentType:       aws.String(f.ContentType),
				CacheControl:      aws.String(fmt.Sprintf("public, max-age=%d", f.CacheSeconds)),
				ContentEncoding:   aws.String(f.ContentEncoding),
				CopySource:        aws.String(joinPath(domain, f.Source)),
				ACL:               aws.String(s3.BucketCannedACLPublicRead),
			})

			return err
		},

		ListBucketFilepaths: func(path string) ([]string, error) {
			list, err := s3Session.ListObjectsV2(&s3.ListObjectsV2Input{
				Prefix:  aws.String(path),
				Bucket:  aws.String(domain),
				MaxKeys: aws.Int64(1000),
			})
			if err != nil {
				return []string{}, err
			}

			if *list.IsTruncated {
				return []string{}, errors.New(fmt.Sprintf("More than %d HTML files in version, rollback is not supported.  Consider filing a GitHub issue if you need support for this.", list.MaxKeys))
			}

			filepaths := make([]string, len(list.Contents))
			for i, file := range list.Contents {
				filepaths[i] = *file.Key
			}

			return filepaths, nil
		},
	}, nil
}

func joinPath(parts ...string) string {
	// Like filepath.Join, but always uses '/'
	out := filepath.Join(parts...)

	if os.PathSeparator != '/' {
		out = strings.Replace(out, string(os.PathSeparator), "/", -1)
	}

	return out
}
