package fs

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
)

func CreateFS(gclient *storage.Client, ctx context.Context, domain string, projectID string, location string) error {
	bucket := gclient.Bucket(domain)
	err := bucket.Create(ctx, projectID, &storage.BucketAttrs{
		Name: domain,
		DefaultObjectACL: []storage.ACLRule{
			{
				Entity: storage.AllUsers,
				Role:   storage.RoleReader,
			},
		},
		Location: location,
		Website: &storage.BucketWebsite{
			MainPageSuffix: "index.html",
			NotFoundPage:   "404.html",
		},
	})
	if err != nil {
		return err
	}

	fmt.Printf("Your FS bucket has been created, and can be edited at https://console.cloud.google.com/storage/browser?project=%s!", projectID)
	return nil
}
