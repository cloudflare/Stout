package types

type UploadFileHolder struct {
	Dest         string
	Body         []byte
	MD5          []byte
	CacheSeconds int
	ContentType  string
}

type CopyFileHolder struct {
	Source       string
	Dest         string
	CacheSeconds int
	ContentType  string
}

type FSProviderFunctions struct {
	UploadFile          func(f UploadFileHolder) error
	CopyFile            func(f CopyFileHolder) error
	ListBucketFilepaths func() (filepaths []string, err error)
}
