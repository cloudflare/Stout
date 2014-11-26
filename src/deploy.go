package main

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"mime"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"code.google.com/p/go.net/html"

	"log"

	"github.com/crowdmob/goamz/s3"
	"github.com/wsxiaoys/terminal/color"
)

const (
	SCRIPT = iota
	STYLE
)

const UPLOAD_WORKERS = 20

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

func hashBytes(data []byte) []byte {
	hash := md5.New()
	must(io.Copy(hash, bytes.NewReader(data)))
	return hash.Sum(nil)
}

func hashFiles(files []string) string {
	hash := new(big.Int)
	for _, file := range files {
		val := new(big.Int)
		val.SetBytes(hashFile(file))

		hash = hash.Xor(hash, val)
	}

	return fmt.Sprintf("%x", hash)
}

func getRef() string {
	gitPath := mustString(exec.LookPath("git"))

	cmd := exec.Command(gitPath, "rev-parse", "--verify", "HEAD")

	out := bytes.Buffer{}
	cmd.Stdout = &out
	panicIf(cmd.Run())

	return string(out.Bytes())
}

func guessContentType(file string) string {
	return mime.TypeByExtension(filepath.Ext(file))
}

func uploadFile(bucket *s3.Bucket, reader io.Reader, dest string, includeHash bool, caching int) string {
	buffer := bytes.NewBuffer([]byte{})
	writer := gzip.NewWriter(buffer)
	must(io.Copy(writer, reader))
	writer.Close()

	data := buffer.Bytes()

	hash := hashBytes(data)
	hashPrefix := fmt.Sprintf("%x", hash)[:12]
	s3Opts := s3.Options{
		ContentMD5:      base64.StdEncoding.EncodeToString(hash),
		ContentEncoding: "gzip",
		CacheControl:    fmt.Sprintf("public, max-age=%d", caching),
	}

	if includeHash {
		dest = filepath.Join(hashPrefix, dest)
	}

	log.Printf("Uploading to %s in %s (%s) [%d]\n", dest, bucket.Name, hashPrefix, caching)
	err := bucket.PutReader(dest, buffer, int64(len(data)), guessContentType(dest), s3.PublicRead, s3Opts)
	panicIf(err)

	return dest
}

type FileRef struct {
	LocalPath    string
	RemotePath   string
	UploadedPath string
}

type FileInst struct {
	File     *FileRef
	InstPath string
}

func writeFiles(options Options, includeHash bool, files chan *FileRef) {
	bucket := s3Session.Bucket(options.Bucket)

	for file := range files {
		handle := must(os.Open(file.LocalPath)).(*os.File)
		defer handle.Close()

		var ttl int
		ttl = FOREVER
		if !includeHash {
			ttl = LIMITED
		}

		(*file).UploadedPath = uploadFile(bucket, handle, file.RemotePath, includeHash, ttl)
	}
}

func deployFiles(options Options, includeHash bool, files []*FileRef) {
	ch := make(chan *FileRef)

	wg := new(sync.WaitGroup)
	for i := 0; i < UPLOAD_WORKERS; i++ {
		wg.Add(1)
		go func() {
			writeFiles(options, includeHash, ch)
			wg.Done()
		}()
	}

	for _, file := range files {
		if !includeHash && strings.HasSuffix(file.RemotePath, ".html") {
			panic(fmt.Sprintf("Cowardly refusing to deploy an html file (%s) without versioning.  Add the file to the --html list to deploy with versioning.", file.RemotePath))
		}

		ch <- file
	}

	close(ch)

	wg.Wait()
}

func addFiles(form uint8, parent *html.Node, files []string) {
	for _, file := range files {
		node := html.Node{
			Type: html.ElementNode,
		}
		switch form {
		case SCRIPT:
			node.Data = "script"
			node.Attr = []html.Attribute{
				html.Attribute{
					Key: "src",
					Val: file,
				},
			}

		case STYLE:
			node.Data = "link"
			node.Attr = []html.Attribute{
				html.Attribute{
					Key: "rel",
					Val: "stylesheet",
				},
				html.Attribute{
					Key: "href",
					Val: file,
				},
			}
		default:
			panic("Type not understood")
		}

		parent.AppendChild(&node)
	}
}

func isLocal(href string) bool {
	parsed := must(url.Parse(href)).(*url.URL)
	return parsed.Host == ""
}

func formatHref(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func renderHTML(options Options, file HTMLFile) string {
	handle := must(os.Open(file.File.LocalPath)).(*os.File)
	defer handle.Close()

	doc := must(html.Parse(handle)).(*html.Node)

	var f func(*html.Node)
	f = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}

		if n.Type == html.ElementNode {
			switch n.Data {
			case "script":
				for i, a := range n.Attr {
					if a.Key == "src" {
						for _, dep := range file.Deps {
							if dep.InstPath == a.Val {
								n.Attr[i].Val = formatHref(dep.File.UploadedPath)
								break
							}
						}
					}
				}
			case "link":
				stylesheet := false
				for _, a := range n.Attr {
					if a.Key == "rel" {
						stylesheet = a.Val == "stylesheet"
						break
					}
				}
				if !stylesheet {
					return
				}

				for i, a := range n.Attr {
					if a.Key == "href" {
						for _, dep := range file.Deps {
							if dep.InstPath == a.Val {
								n.Attr[i].Val = formatHref(dep.File.UploadedPath)
								break
							}
						}
					}
				}
			}
		}
	}
	f(doc)

	buf := bytes.NewBuffer([]byte{})
	panicIf(html.Render(buf, doc))

	return buf.String()
}

func parseHTML(options Options, path string) (files []string, base string) {
	files = make([]string, 0)

	handle := must(os.Open(path)).(*os.File)
	defer handle.Close()

	doc := must(html.Parse(handle)).(*html.Node)

	var f func(*html.Node)
	f = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}

		if n.Type == html.ElementNode {
			switch n.Data {
			case "base":
				for _, a := range n.Attr {
					if a.Key == "href" {
						base = a.Val
					}
				}
			case "script":
				for _, a := range n.Attr {
					if a.Key == "src" {
						if isLocal(a.Val) {
							files = append(files, a.Val)
						}
					}
				}
			case "link":
				local := false
				stylesheet := false
				href := ""
				for _, a := range n.Attr {
					switch a.Key {
					case "href":
						local = isLocal(a.Val)
						href = a.Val
					case "rel":
						stylesheet = a.Val == "stylesheet"
					}
				}
				if local && stylesheet {
					files = append(files, href)
				}
			}
		}
	}
	f(doc)

	return
}

func deployHTML(options Options, id string, file HTMLFile) {
	data := renderHTML(options, file)

	internalPath, err := filepath.Rel(options.Root, file.File.LocalPath)
	if err != nil {
		panic(err)
	}

	permPath := filepath.Join(options.Dest, id, internalPath)
	curPath := filepath.Join(options.Dest, internalPath)

	bucket := s3Session.Bucket(options.Bucket)
	uploadFile(bucket, strings.NewReader(data), permPath, false, FOREVER)

	log.Println("Copying", permPath, "to", curPath)
	copyFile(bucket, permPath, curPath, "text/html", LIMITED)
}

func expandFiles(root string, glob string) []string {
	out := make([]string, 0)
	cases := strings.Split(glob, ",")

	for _, pattern := range cases {
		list := must(filepath.Glob(filepath.Join(root, pattern))).([]string)

		for _, file := range list {
			info := must(os.Stat(file)).(os.FileInfo)

			if info.IsDir() {
				filepath.Walk(file, func(path string, info os.FileInfo, err error) error {
					panicIf(err)

					if !info.IsDir() {
						out = append(out, path)
					}

					return nil
				})
			} else {
				out = append(out, file)
			}
		}
	}
	return out
}

func listFiles(options Options) []*FileRef {
	filePaths := expandFiles(options.Root, options.Files)

	files := make([]*FileRef, len(filePaths))
	for i, path := range filePaths {
		remotePath := filepath.Join(options.Dest, mustString(filepath.Rel(options.Root, path)))

		files[i] = &FileRef{
			LocalPath:  path,
			RemotePath: remotePath,
		}
	}

	return files
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

func extractFileList(options Options, pattern string) (files []string) {
	files = make([]string, 0)

	parts := strings.Split(pattern, ",")

	for _, part := range parts {
		matches, err := filepath.Glob(filepath.Join(options.Root, part))
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
	File FileRef
	Deps []FileInst
	Base string
}

func (f HTMLFile) GetLocalPath() string {
	return f.File.LocalPath
}

func Deploy(options Options) {
	if s3Session == nil {
		s3Session = openS3(options.AWSKey, options.AWSSecret)
	}

	files := listFiles(options)

	htmlFileRefs := filesWithExtension(files, ".html")

	inclFiles := make(map[string]*FileRef)
	htmlFiles := make([]HTMLFile, len(htmlFileRefs))
	for i, file := range htmlFileRefs {
		dir := filepath.Dir(file.LocalPath)

		rel, err := filepath.Rel(options.Root, dir)
		if err != nil {
			panic(err)
		}

		paths, base := parseHTML(options, file.LocalPath)

		if strings.HasPrefix(strings.ToLower(base), "http") || strings.HasPrefix(base, "//") {
			panic("Absolute base tags are not supported")
		}

		htmlFiles[i] = HTMLFile{
			File: *file,
			Deps: make([]FileInst, len(paths)),
			Base: base,
		}

		for j, path := range paths {
			local := filepath.Join(options.Root, rel, base, path)
			remote := filepath.Join(options.Dest, rel, base, path)

			ref, ok := inclFiles[local]
			if !ok {
				ref = &FileRef{
					LocalPath:  local,
					RemotePath: remote,

					// Filled in after the deploy:
					UploadedPath: "",
				}

				inclFiles[local] = ref
			}

			use := FileInst{
				File:     ref,
				InstPath: path,
			}

			htmlFiles[i].Deps[j] = use
		}
	}

	inclFileList := make([]*FileRef, len(inclFiles))
	i := 0
	for _, ref := range inclFiles {
		inclFileList[i] = ref
		i++
	}

	hashPaths := make([]string, 0)
	for _, item := range inclFileList {
		hashPaths = append(hashPaths, item.LocalPath)
	}
	for _, item := range htmlFiles {
		hashPaths = append(hashPaths, item.File.LocalPath)
	}

	hash := hashFiles(hashPaths)
	id := hash[:12]

	deployFiles(options, true, inclFileList)
	deployFiles(options, false, ignoreFiles(files, htmlFileRefs))

	// Ensure that the new files exist in s3
	// Time based on "Eventual Consistency: How soon is eventual?"
	time.Sleep(1500 * time.Millisecond)

	wg := sync.WaitGroup{}
	for _, file := range htmlFiles {
		wg.Add(1)

		go func(file HTMLFile) {
			defer wg.Done()
			deployHTML(options, id, file)
		}(file)
	}

	wg.Wait()

	color.Printf(`
+------------------------------------+
|         @{g}Deploy Successful!@{|}         |
|                                    |
|       Deploy ID: @{?}%s@{|}      |
+------------------------------------+
`, id)

}

func deployCmd() {
	options, _ := parseOptions()
	loadConfigFile(&options)

	if options.Bucket == "" {
		panic("You must specify a bucket")
	}

	if options.AWSKey == "" || options.AWSSecret == "" {
		panic("You must specify your AWS credentials")
	}

	Deploy(options)
}
