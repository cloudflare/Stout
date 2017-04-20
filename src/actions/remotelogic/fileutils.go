package remotelogic

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"math/big"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/eagerio/Stout/src/utils"
)

const (
	LIMITED = 60
	FOREVER = 31556926
)

// Merge files using forward slashes and not the system path seperator if that is different
// Useful since windows has backslash path separators instead of forward slash which is hard to use with S3
func joinPath(parts ...string) string {
	// Like filepath.Join, but always uses '/'
	out := filepath.Join(parts...)

	if os.PathSeparator != '/' {
		out = strings.Replace(out, string(os.PathSeparator), "/", -1)
	}

	return out
}

// Create a md5 hash from the bytes given
func hashBytes(data []byte) []byte {
	hash := md5.New()
	utils.Must(io.Copy(hash, bytes.NewReader(data)))
	return hash.Sum(nil)
}

// Create the hash of the filepath given
func hashFilepath(path string) []byte {
	hash := md5.New()
	io.WriteString(hash, path)
	io.WriteString(hash, "\n")

	// TODO: Encode type?

	ref := utils.Must(os.Open(path)).(*os.File)
	defer ref.Close()

	utils.Must(io.Copy(hash, ref))

	return hash.Sum(nil)
}

// Create a hash from the xors of the filepaths of files
// useful for creating hashes of a folder/set of files
func hashFilepaths(files []string) string {
	hash := new(big.Int)
	for _, file := range files {
		val := new(big.Int)
		val.SetBytes(hashFilepath(file))

		hash = hash.Xor(hash, val)
	}

	return fmt.Sprintf("%x", hash)
}

// Guess the correct mime type for the file extension
func guessContentType(file string) string {
	return mime.TypeByExtension(filepath.Ext(file))
}

// Should this file compress? use the NO_GZIP list to check if the extension is available for compression
func shouldCompress(file string) bool {
	ext := filepath.Ext(file)

	for _, e := range []string{"mp4", "webm", "ogg"} {
		if "."+e == ext {
			return false
		}
	}

	return true
}

// Returns if the host is blank
func isLocal(href string) bool {
	parsedUrl := utils.Must(url.Parse(href)).(*url.URL)
	return parsedUrl.Host == ""
}

// add forward slash prefix to links
func formatHref(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

// Get file references from the options
func listFiles(root string, files string, dest string) []*FileRef {
	filePaths := expandFiles(root, files)

	fileObjects := make([]*FileRef, len(filePaths))
	for i, path := range filePaths {
		remotePath := joinPath(dest, utils.MustString(filepath.Rel(root, path)))

		for strings.HasPrefix(remotePath, "../") {
			remotePath = remotePath[3:]
		}

		fileObjects[i] = &FileRef{
			LocalPath:  path,
			RemotePath: remotePath,
		}
	}

	return fileObjects
}

func ignoreFiles(full []*FileRef, rem []*FileRef) []*FileRef {
	out := make([]*FileRef, 0, len(full))

	for _, file := range full {
		ignore := false
		path := filepath.Clean(file.LocalPath)

		for _, remFile := range rem {
			if filepath.Clean(remFile.LocalPath) == path {
				ignore = true
				break
			}
		}

		if !ignore {
			out = append(out, file)
		}
	}

	return out
}

func extractFileList(root string, pattern string) (files []string) {
	files = make([]string, 0)

	parts := strings.Split(pattern, ",")

	for _, part := range parts {
		matches, err := filepath.Glob(joinPath(root, part))
		if err != nil {
			panic(err)
		}
		if matches == nil {
			panic(fmt.Sprintf("Pattern %s did not match any files", part))
		}

		files = append(files, matches...)
	}

	return files
}

// Pick out files with a specific extension
func filesWithExtension(files []*FileRef, ext string) (outFiles []*FileRef) {
	outFiles = make([]*FileRef, 0)
	for _, file := range files {
		if filepath.Ext(file.LocalPath) == ext {
			outFiles = append(outFiles, file)
		}
	}

	return
}

type HTMLFile struct {
	File         FileRef
	Dependencies []FileInst
	Base         string
}

func (f HTMLFile) GetLocalPath() string {
	return f.File.LocalPath
}
