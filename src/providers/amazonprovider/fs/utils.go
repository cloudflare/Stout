package fs

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

	"github.com/zackbloom/goamz/s3"
)

const (
	LIMITED = 60
	FOREVER = 31556926
)

// Catch errors and panic if there is an error
func panicIf(err error) {
	if err != nil {
		panic(err)
	}
}

// Catch errors and panic if there is an error
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

// Copy file in s3
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

// Create the hash of the filepath given
func hashFile(path string) []byte {
	hash := md5.New()
	io.WriteString(hash, path)
	io.WriteString(hash, "\n")

	// TODO: Encode type?

	ref := must(os.Open(path)).(*os.File)
	defer ref.Close()

	must(io.Copy(hash, ref))

	return hash.Sum(nil)
}

// Create a md5 hash from the bytes given
func hashBytes(data []byte) []byte {
	hash := md5.New()
	must(io.Copy(hash, bytes.NewReader(data)))
	return hash.Sum(nil)
}

// Create a hash from the xors of the filepaths of files
// useful for creating hashes of a folder/set of files
func hashFiles(files []string) string {
	hash := new(big.Int)
	for _, file := range files {
		val := new(big.Int)
		val.SetBytes(hashFile(file))

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
	for _, e := range NO_GZIP {
		if "."+e == ext {
			return false
		}
	}

	return true
}

// Returns if the host is blank
func isLocal(href string) bool {
	parsed := must(url.Parse(href)).(*url.URL)
	return parsed.Host == ""
}

// add forward slash prefix to links
func formatHref(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}
