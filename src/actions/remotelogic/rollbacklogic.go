package remotelogic

import (
	"log"
	"path/filepath"
	"sync"

	"code.cfops.it/apps/proxy/utils"

	"github.com/eagerio/Stout/src/types"
)

// Go go the version prefix folder on s3 and copy the html files over to the root as the currently active files
func Rollback(fsFuncs types.FSProviderFunctions, g types.GlobalFlags, r types.RollbackFlags) error {
	dest := r.Dest
	version := r.Version

	prefix := filepath.Join(dest, version) + "/"

	wait := sync.WaitGroup{}
	count := 0

	filepaths, err := fsFuncs.ListBucketFilepaths()
	if err != nil {
		return err
	}

	for _, path := range filepaths {
		wait.Add(1)
		go func(path string) {
			defer wait.Done()

			if filepath.Ext(path) != ".html" {
				log.Printf("Skipping non-html file %s", path)
				return
			}

			newPath := filepath.Join(dest, path[len(prefix):])

			log.Printf("Aliasing %s to %s", path, newPath)

			//replace old files with new prefixed files in root
			utils.PanicIf(fsFuncs.CopyFile(types.CopyFileHolder{
				Source:       path,
				Dest:         newPath,
				CacheSeconds: LIMITED,
				ContentType:  "text/html",
			}))
			count++
		}(path)
	}

	wait.Wait()

	log.Printf("Reverted %d HTML files to version %s", count, version)
	return nil
}
