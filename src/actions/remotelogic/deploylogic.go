package remotelogic

import (
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cenk/backoff"
	"github.com/eagerio/Stout/src/types"
	"github.com/eagerio/Stout/src/utils"
	"golang.org/x/net/html"

	"log"

	"github.com/wsxiaoys/terminal/color"
)

const (
	SCRIPT = iota
	STYLE
)

const UPLOAD_WORKERS = 20

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

type UploadFileRequest struct {
	Bucket       string
	Reader       io.Reader
	Path         string
	Dest         string
	IncludeHash  bool
	CacheSeconds int
}

func uploadFileToProvider(fsFuncs types.FSProviderFunctions, req UploadFileRequest) (remotePath string) {
	buffer := bytes.NewBuffer([]byte{})

	compress := shouldCompress(req.Path)

	if compress {
		writer := gzip.NewWriter(buffer)
		utils.Must(io.Copy(writer, req.Reader))
		writer.Close()
	} else {
		utils.Must(io.Copy(buffer, req.Reader))
	}

	data := buffer.Bytes()

	hash := hashBytes(data)
	hashPrefix := fmt.Sprintf("%x", hash)[:12]

	dest := req.Path
	if req.IncludeHash {
		dest = hashPrefix + "_" + dest
	}
	dest = filepath.Join(req.Dest, dest)

	log.Printf("Uploading to %s in %s (%s) [%d]\n", dest, req.Bucket, hashPrefix, req.CacheSeconds)

	op := func() error {
		return fsFuncs.UploadFile(types.UploadFileHolder{
			Dest:         dest,
			Body:         data,
			MD5:          hash,
			CacheSeconds: req.CacheSeconds,
			ContentType:  guessContentType(req.Dest),
		})
	}

	back := backoff.NewExponentialBackOff()
	back.MaxElapsedTime = 30 * time.Second

	err := backoff.RetryNotify(op, back, func(err error, next time.Duration) {
		log.Println("Error uploading", err, "retrying in", next)
	})
	utils.PanicIf(err)

	return dest
}

// Open files and pass the handle to uploadFileToProvider function
func writeFiles(fsFuncs types.FSProviderFunctions, domain string, dest string, includeHash bool, files chan *FileRef) {
	for file := range files {
		handle := utils.Must(os.Open(file.LocalPath)).(*os.File)
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

		(*file).UploadedPath = uploadFileToProvider(fsFuncs, UploadFileRequest{
			Bucket:       domain,
			Reader:       handle,
			Path:         partialPath,
			Dest:         dest,
			IncludeHash:  includeHash,
			CacheSeconds: ttl,
		})
	}
}

// Deploy/upload files consurently
func deployFiles(fsFuncs types.FSProviderFunctions, domain string, dest string, includeHash bool, files []*FileRef) {
	ch := make(chan *FileRef)

	wait := new(sync.WaitGroup)
	for i := 0; i < UPLOAD_WORKERS; i++ {
		wait.Add(1)
		go func() {
			writeFiles(fsFuncs, domain, dest, includeHash, ch)
			wait.Done()
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

	wait.Wait()
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
	handle := utils.Must(os.Open(file.File.LocalPath)).(*os.File)
	defer handle.Close()

	doc := utils.Must(html.Parse(handle)).(*html.Node)

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
						for _, dep := range file.Dependencies {
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
						for _, dep := range file.Dependencies {
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
	utils.PanicIf(html.Render(buf, doc))

	return buf.String()
}

// parse html files
// returns slice of files found in the html files and a base string for the base
func parseHTML(path string) (files []string, base string) {
	files = make([]string, 0)

	handle := utils.Must(os.Open(path)).(*os.File)
	defer handle.Close()

	doc := utils.Must(html.Parse(handle)).(*html.Node)

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
func deployHTML(fsFuncs types.FSProviderFunctions, domain string, root string, dest string, id string, file HTMLFile) {
	data := renderHTML(file)

	internalPath, err := filepath.Rel(root, file.File.LocalPath)
	if err != nil {
		panic(err)
	}

	fromPath := joinPath(dest, id, internalPath)
	toPath := joinPath(dest, internalPath)

	uploadFileToProvider(fsFuncs, UploadFileRequest{
		Bucket:       domain,
		Reader:       strings.NewReader(data),
		Dest:         fromPath,
		IncludeHash:  false,
		CacheSeconds: FOREVER,
	})

	log.Println("Copying", fromPath, "to", toPath)

	utils.PanicIf(fsFuncs.CopyFile(types.CopyFileHolder{
		Source:       fromPath,
		Dest:         toPath,
		ContentType:  "text/html; charset=utf-8",
		CacheSeconds: LIMITED,
	}))
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

		list := utils.Must(filepath.Glob(pattern)).([]string)

		for _, file := range list {
			info := utils.Must(os.Stat(file)).(os.FileInfo)

			if info.IsDir() {
				filepath.Walk(file, func(path string, info os.FileInfo, err error) error {
					utils.PanicIf(err)

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

// Deploy main function
func Deploy(fsFuncs types.FSProviderFunctions, g types.GlobalFlags, d types.DeployFlags) error {
	domain := g.Domain
	root := d.Root
	files := d.Files
	dest := d.Dest

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

			if strings.HasPrefix(strings.ToLower(base), "http") || strings.HasPrefix(base, "//") {
				return errors.New("Absolute base tags are not supported in Stout.")
			}

			if strings.HasSuffix(base, "/") {
				base = base[:len(base)-1]
			}

			htmlFiles[i] = HTMLFile{
				File:         *file,
				Dependencies: make([]FileInst, len(paths)),
				Base:         base,
			}

			dest := joinPath(dest, base)
			realRoot := joinPath(root, base)

			if strings.HasPrefix(base, "/") {
				if strings.HasPrefix(base, "/"+dest) {
					dest = base
				}

				if strings.HasSuffix(root, base) {
					realRoot = root
				}
			}

			for j, path := range paths {
				// get file from local path, put file in remote path
				var local, remote string
				if strings.HasPrefix(path, "/") { // absolute link to bucket root
					local = joinPath(root, path)
					remote = joinPath(dest, path)
				} else { // relative url (which can be affected by base tags)
					if strings.HasPrefix(base, "/") { // base tag absolute to bucket root
						local = joinPath(realRoot, path)
						remote = joinPath(dest, path)
					} else { // base tag relative to current folder
						local = joinPath(root, rel, base, path)
						remote = joinPath(dest, rel, base, path)
					}
				}

				// notice `for` and not `if`
				for strings.HasPrefix(remote, "../") {
					remote = remote[3:]
				}

				//check if the file is already included elsewhere
				ref, ok := inclFiles[local]
				if !ok {
					ref = &FileRef{
						LocalPath:    local,
						RemotePath:   remote,
						UploadedPath: "", // Filled in after the deploy:
					}
					inclFiles[local] = ref
				}

				htmlFiles[i].Dependencies[j] = FileInst{
					File:     ref,
					InstPath: path,
				}
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
		filepaths := make([]string, 0)
		for _, item := range inclFileList {
			filepaths = append(filepaths, item.LocalPath)
		}
		for _, item := range htmlFiles {
			filepaths = append(filepaths, item.File.LocalPath)
		}
		hash := hashFilepaths(filepaths)
		id = hash[:12] //this will go to the html folder

		deployFiles(fsFuncs, domain, dest, true, inclFileList)
	}
	//deploy all files requested except the html files
	deployFiles(fsFuncs, domain, dest, false, ignoreFiles(fileObjects, htmlFileRefs))

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
				deployHTML(fsFuncs, domain, root, dest, id, file)
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
