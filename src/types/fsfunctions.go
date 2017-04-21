package types

type UploadFileHolder struct {
	Dest            string
	Body            []byte
	MD5             []byte
	CacheSeconds    int
	ContentType     string
	ContentEncoding string
}

type CopyFileHolder struct {
	Source          string
	Dest            string
	CacheSeconds    int
	ContentType     string
	ContentEncoding string
}

type FSProviderFunctions struct {
	UploadFile          func(f UploadFileHolder) error
	CopyFile            func(f CopyFileHolder) error
	ListBucketFilepaths func(path string) (filepaths []string, err error)
}
