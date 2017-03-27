package fs

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
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

// File reference
type FileRef struct {
	LocalPath    string
	RemotePath   string
	UploadedPath string // uploaded path includes the hash
}

// Instance of a file reference
type FileInst struct {
	File     *FileRef
	InstPath string
}

// Open files and pass the handle to uploadFile function
func writeFiles(s3Session *s3.S3, domain string, dest string, includeHash bool, files chan *FileRef) {
	bucket := s3Session.Bucket(domain)

	for file := range files {
		handle := must(os.Open(file.LocalPath)).(*os.File)
		defer handle.Close()

		// The presence of hash determines the expiration
		var ttl int
		ttl = FOREVER
		if !includeHash {
			ttl = LIMITED
		}

		remote := file.RemotePath
		if strings.HasPrefix(remote, "/") {
			remote = remote[1:]
		}
		partialPath, err := filepath.Rel(dest, remote)
		if err != nil {
			panic(err)
		}

		(*file).UploadedPath = uploadFile(UploadFileRequest{
			Bucket:       bucket,
			Reader:       handle,
			Path:         partialPath,
			Dest:         dest,
			IncludeHash:  includeHash,
			CacheSeconds: ttl,
		})
	}
}

// Deploy/upload files consurently
func deployFiles(s3Session *s3.S3, domain string, dest string, includeHash bool, files []*FileRef) {
	ch := make(chan *FileRef)

	wg := new(sync.WaitGroup)
	for i := 0; i < UPLOAD_WORKERS; i++ {
		wg.Add(1)
		go func() {
			writeFiles(s3Session, domain, dest, includeHash, ch)
			wg.Done()
		}()
	}

	for _, file := range files {
		//catch the case where hash might not have been supplied?
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

// Render HTML file
func renderHTML(file HTMLFile) string {
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
						//find the link from the dependencies and replace
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
				// TODO(renandincer): take a second look here
				// This node is not a stylesheet
				if !stylesheet {
					return
				}

				// If it is a link replace
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

// parse html files
// returns slice of files found in the html files and a base string for the base
func parseHTML(path string) (files []string, base string) {
	files = make([]string, 0)

	handle := must(os.Open(path)).(*os.File)
	defer handle.Close()

	doc := must(html.Parse(handle)).(*html.Node)

	var f func(*html.Node)
	// loop to go through all nodes of the html file
	f = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
		if n.Type == html.ElementNode {
			//if it is a base?
			switch n.Data { //switch by tag name
			case "base":
				for _, a := range n.Attr {
					if a.Key == "href" {
						base = a.Val
					}
				}
			// or a script with src attribute
			case "script":
				for _, a := range n.Attr {
					if a.Key == "src" {
						if isLocal(a.Val) {
							files = append(files, a.Val) //add local files to queue
						}
					}
				}
			//or link attibute
			case "link":
				local := false
				stylesheet := false
				href := ""
				for _, a := range n.Attr {
					switch a.Key {
					case "href":
						local = isLocal(a.Val) //determine if the link is local (aka. without a host)
						href = a.Val
					case "rel":
						stylesheet = a.Val == "stylesheet"
					}
				}
				if local && stylesheet {
					files = append(files, href) //if both local and stylesheet add to files
				}
			}
		}
	}
	f(doc)

	return
}

// deploy html to its permanent hashed path and copy them outside for public
func deployHTML(s3Session *s3.S3, domain string, root string, dest string, id string, file HTMLFile) {
	data := renderHTML(file)

	internalPath, err := filepath.Rel(root, file.File.LocalPath)
	if err != nil {
		panic(err)
	}

	permPath := joinPath(dest, id, internalPath)
	curPath := joinPath(dest, internalPath)

	bucket := s3Session.Bucket(domain)
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

// List all files to be acted upon from the root and glob patterns
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

// Get file references from the options
func listFiles(root string, files string, dest string) []*FileRef {
	filePaths := expandFiles(root, files)

	fileObjects := make([]*FileRef, len(filePaths))
	for i, path := range filePaths {
		remotePath := joinPath(dest, mustString(filepath.Rel(root, path)))

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
	File FileRef
	Deps []FileInst
	Base string
}

func (f HTMLFile) GetLocalPath() string {
	return f.File.LocalPath
}

// Deploy main function
func Deploy(s3Session *s3.S3, domain string, root string, files string, dest string) error {
	// list all files that match the glob pattern in the root
	fileObjects := listFiles(root, files, dest)

	htmlFileRefs := filesWithExtension(fileObjects, ".html")
	var htmlFiles []HTMLFile //slice with html files
	var id string

	if len(htmlFileRefs) == 0 {
		log.Println("No HTML files found")
	} else {
		inclFiles := make(map[string]*FileRef)
		htmlFiles = make([]HTMLFile, len(htmlFileRefs)) //slice with html files

		for i, file := range htmlFileRefs {
			dir := filepath.Dir(file.LocalPath)

			rel, err := filepath.Rel(root, dir) //get relative filepath
			if err != nil {
				panic(err)
			}

			// get a slice of all paths to stylesheets and scripts
			// get base if there is a base tag to set the default target for all links
			paths, base := parseHTML(file.LocalPath)

			// TODO(renandincer): make this error more clear
			// https is included in the http prefix :)
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
			if strings.HasPrefix(base, "/") && strings.HasPrefix(base, "/"+dest) {
				dest = base
			} else {
				dest = joinPath(dest, base)
			}

			var realRoot string
			if strings.HasPrefix(base, "/") && strings.HasSuffix(root, base) {
				realRoot = root
			} else {
				realRoot = joinPath(root, base)
			}

			for j, path := range paths {
				var local, remote string
				//put file locations in dest and root
				if strings.HasPrefix(path, "/") {
					local = joinPath(root, path)
					remote = joinPath(dest, path)
				} else {
					if strings.HasPrefix(base, "/") {
						local = joinPath(realRoot, path)
						remote = joinPath(dest, path)
					} else {
						local = joinPath(root, rel, base, path)
						remote = joinPath(dest, rel, base, path)
					}
				}
				//TODO(renandincer): would this work if the reference is two levels down?
				for strings.HasPrefix(remote, "../") {
					remote = remote[3:]
				}

				//check if the file is already included elsewhere
				ref, ok := inclFiles[local]
				if !ok {
					ref = &FileRef{
						LocalPath:  local,
						RemotePath: remote,

						// Filled in after the deploy:
						UploadedPath: "",
					}
					// if not add it
					inclFiles[local] = ref
				}

				use := FileInst{
					File:     ref,
					InstPath: path,
				}

				htmlFiles[i].Deps[j] = use
			}
		}

		//convert the inclFile map to list
		inclFileList := make([]*FileRef, len(inclFiles))
		i := 0
		for _, ref := range inclFiles {
			inclFileList[i] = ref
			i++
		}

		// hash all the paths of all html files and dependencies together
		hashPaths := make([]string, 0)
		for _, item := range inclFileList {
			hashPaths = append(hashPaths, item.LocalPath)
		}
		for _, item := range htmlFiles {
			hashPaths = append(hashPaths, item.File.LocalPath)
		}
		hash := hashFiles(hashPaths)
		id = hash[:12] //this will go to the html folder

		deployFiles(s3Session, domain, dest, true, inclFileList)
	}
	//deploy all files requested except the html files
	deployFiles(s3Session, domain, dest, false, ignoreFiles(fileObjects, htmlFileRefs))

	//deploy html
	if len(htmlFileRefs) != 0 {
		// Ensure that the new files exist in s3
		// Time based on "Eventual Consistency: How soon is eventual?"
		time.Sleep(1500 * time.Millisecond)

		wg := sync.WaitGroup{}
		for _, file := range htmlFiles {
			wg.Add(1)

			go func(file HTMLFile) {
				defer wg.Done()
				deployHTML(s3Session, domain, root, dest, id, file)
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

	return nil
}
