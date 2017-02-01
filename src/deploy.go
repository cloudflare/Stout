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

	"github.com/cenk/backoff"
	"golang.org/x/net/html"

	"log"

	"github.com/wsxiaoys/terminal/color"
	"github.com/zackbloom/goamz/s3"
)

const (
	SCRIPT = iota
	STYLE
)

const UPLOAD_WORKERS = 20

var NO_GZIP = []string{
	"mp4",
	"webm",
	"ogg",
}

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

func shouldCompress(file string) bool {
	ext := filepath.Ext(file)
	for _, e := range NO_GZIP {
		if "."+e == ext {
			return false
		}
	}

	return true
}

type UploadFileRequest struct {
	Bucket       *s3.Bucket
	Reader       io.Reader
	Path         string
	Dest         string
	IncludeHash  bool
	CacheSeconds int
}

func uploadFile(req UploadFileRequest) (remotePath string) {
	buffer := bytes.NewBuffer([]byte{})

	compress := shouldCompress(req.Path)

	if compress {
		writer := gzip.NewWriter(buffer)
		must(io.Copy(writer, req.Reader))
		writer.Close()
	} else {
		must(io.Copy(buffer, req.Reader))
	}

	data := buffer.Bytes()

	hash := hashBytes(data)
	hashPrefix := fmt.Sprintf("%x", hash)[:12]
	s3Opts := s3.Options{
		ContentMD5:   base64.StdEncoding.EncodeToString(hash),
		CacheControl: fmt.Sprintf("public, max-age=%d", req.CacheSeconds),
	}

	if compress {
		s3Opts.ContentEncoding = "gzip"
	}

	dest := req.Path
	if req.IncludeHash {
		dest = hashPrefix + "_" + dest
	}
	dest = filepath.Join(req.Dest, dest)

	log.Printf("Uploading to %s in %s (%s) [%d]\n", dest, req.Bucket.Name, hashPrefix, req.CacheSeconds)

	op := func() error {
		// We need to create a new reader each time, as we might be doing this more than once (if it fails)
		return req.Bucket.PutReader(dest, bytes.NewReader(data), int64(len(data)), guessContentType(dest)+"; charset=utf-8", s3.PublicRead, s3Opts)
	}

	back := backoff.NewExponentialBackOff()
	back.MaxElapsedTime = 30 * time.Second

	err := backoff.RetryNotify(op, back, func(err error, next time.Duration) {
		log.Println("Error uploading", err, "retrying in", next)
	})
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

		remote := file.RemotePath
		if strings.HasPrefix(remote, "/") {
			remote = remote[1:]
		}
		partialPath, err := filepath.Rel(options.Dest, remote)
		if err != nil {
			panic(err)
		}

		(*file).UploadedPath = uploadFile(UploadFileRequest{
			Bucket:       bucket,
			Reader:       handle,
			Path:         partialPath,
			Dest:         options.Dest,
			IncludeHash:  includeHash,
			CacheSeconds: ttl,
		})
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
			panic(fmt.Sprintf("Cowardly refusing to deploy an html file (%s) without versioning.", file.RemotePath))
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

	permPath := joinPath(options.Dest, id, internalPath)
	curPath := joinPath(options.Dest, internalPath)

	bucket := s3Session.Bucket(options.Bucket)
	uploadFile(UploadFileRequest{
		Bucket:       bucket,
		Reader:       strings.NewReader(data),
		Path:         permPath,
		IncludeHash:  false,
		CacheSeconds: FOREVER,
	})

	log.Println("Copying", permPath, "to", curPath)
	copyFile(bucket, permPath, curPath, "text/html; charset=utf-8", LIMITED)
}

func expandFiles(root string, glob string) []string {
	out := make([]string, 0)
	cases := strings.Split(glob, ",")

	for _, pattern := range cases {
		if strings.HasPrefix(pattern, "-/") {
			pattern = pattern[2:]
		} else {
			pattern = joinPath(root, pattern)
		}

		list := must(filepath.Glob(pattern)).([]string)

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
		remotePath := joinPath(options.Dest, mustString(filepath.Rel(options.Root, path)))

		for strings.HasPrefix(remotePath, "../") {
			remotePath = remotePath[3:]
		}

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
		matches, err := filepath.Glob(joinPath(options.Root, part))
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
		s3Session = openS3(options.AWSKey, options.AWSSecret, options.AWSRegion)
	}

	files := listFiles(options)

	htmlFileRefs := filesWithExtension(files, ".html")
	var htmlFiles []HTMLFile
	var id string

	if len(htmlFileRefs) == 0 {
		log.Println("No HTML files found")
	} else {
		inclFiles := make(map[string]*FileRef)
		htmlFiles = make([]HTMLFile, len(htmlFileRefs))
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

			if strings.HasSuffix(base, "/") {
				base = base[:len(base)-1]
			}

			htmlFiles[i] = HTMLFile{
				File: *file,
				Deps: make([]FileInst, len(paths)),
				Base: base,
			}

			var dest string
			if strings.HasPrefix(base, "/") && strings.HasPrefix(base, "/"+options.Dest) {
				dest = base
			} else {
				dest = joinPath(options.Dest, base)
			}

			var root string
			if strings.HasPrefix(base, "/") && strings.HasSuffix(options.Root, base) {
				root = options.Root
			} else {
				root = joinPath(options.Root, base)
			}

			for j, path := range paths {
				var local, remote string
				if strings.HasPrefix(path, "/") {
					local = joinPath(options.Root, path)
					remote = joinPath(options.Dest, path)
				} else {
					if strings.HasPrefix(base, "/") {
						local = joinPath(root, path)
						remote = joinPath(dest, path)
					} else {
						local = joinPath(options.Root, rel, base, path)
						remote = joinPath(options.Dest, rel, base, path)
					}
				}

				for strings.HasPrefix(remote, "../") {
					remote = remote[3:]
				}

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
		id = hash[:12]

		deployFiles(options, true, inclFileList)
	}

	deployFiles(options, false, ignoreFiles(files, htmlFileRefs))

	if len(htmlFileRefs) != 0 {
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
	}

	visId := id
	if id == "" {
		visId = "0 HTML Files"
	}

	color.Printf(`
+------------------------------------+
|         @{g}Deploy Successful!@{|}         |
|                                    |
|       Deploy ID: @{?}%s@{|}      |
+------------------------------------+
`, visId)

}

func deployCmd() {
	options, _ := parseOptions()
	loadConfigFile(&options)
	addAWSConfig(&options)

	if options.Bucket == "" {
		panic("You must specify a bucket")
	}

	if options.AWSKey == "" || options.AWSSecret == "" {
		panic("You must specify your AWS credentials")
	}

	Deploy(options)
}
