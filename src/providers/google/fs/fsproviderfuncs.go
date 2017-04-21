package fs

import (
	"context"
	"fmt"

	"google.golang.org/api/iterator"

	"cloud.google.com/go/storage"

	"github.com/eagerio/Stout/src/types"
)

func FSProviderFuncs(gclient *storage.Client, ctx context.Context, domain string) (types.FSProviderFunctions, error) {
	bucket := gclient.Bucket(domain)

	return types.FSProviderFunctions{
		UploadFile: func(f types.UploadFileHolder) error {
			file := bucket.Object(f.Dest)

			w := file.NewWriter(ctx)
			w.ContentType = f.ContentType
			w.ContentEncoding = f.ContentEncoding
			w.CacheControl = fmt.Sprintf("public, max-age=%d", f.CacheSeconds)
			_, err := w.Write(f.Body)
			if err != nil {
				return err
			}

			return w.Close()
		},

		CopyFile: func(f types.CopyFileHolder) error {
			source := bucket.Object(f.Source)
			dest := bucket.Object(f.Dest)

			copier := dest.CopierFrom(source)
			copier.ContentType = f.ContentType
			copier.ContentEncoding = f.ContentEncoding
			copier.CacheControl = fmt.Sprintf("public, max-age=%d", f.CacheSeconds)

			_, err := copier.Run(ctx)
			return err
		},

		ListBucketFilepaths: func(path string) ([]string, error) {
			fileIterator := bucket.Objects(ctx, &storage.Query{
				Prefix: path,
			})

			totalFiles := fileIterator.PageInfo().Remaining()
			filepaths := make([]string, totalFiles)

			for i := 0; i < totalFiles; i++ {
				file, err := fileIterator.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					return []string{}, err
				}

				filepaths[i] = file.Prefix
			}
			return filepaths, nil
		},
	}, nil
}
