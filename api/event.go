package api

type ApiGenImageThumbnailReq struct {
	Identifier string `desc:"identifier"`
	FileId     string `desc:"file id from mini-fstore"`
	ReplyTo    string `desc:"event bus that will receive event about the compressed image (see ImageCompressReplyEvent)."`
}

type ApiGenVideoThumbnailReq struct {
	Identifier string `desc:"identifier"`
	FileId     string `desc:"file id from mini-fstore"`
	ReplyTo    string `desc:"event bus that will receive event about the generated video thumbnail (see GenVideoThumbnailReplyEvent)."`
}

// Event replied from hammer about the compressed image.
type ImageCompressReplyEvent struct {
	Identifier string // identifier
	FileId     string // file id from mini-fstore
}

// Event replied from hammer about the generated video thumbnail.
type GenVideoThumbnailReplyEvent struct {
	Identifier string // identifier
	FileId     string // file id from mini-fstore
}

type UnzipFileReplyEvent struct {
	ZipFileId  string
	ZipEntries []ZipEntry
	Extra      string
}

type ZipEntry struct {
	FileId string
	Md5    string
	Name   string
	Size   int64
}
