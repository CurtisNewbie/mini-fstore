package api

const (
	// event bus to trigger image compression
	CompressImageTriggerEventBus = "event.bus.fstore.image.compress.processing"

	// event bus to trigger video thumbnail generation
	GenVideoThumbnailTriggerEventBus = "event.bus.fstore.video.thumbnail.processing"
)

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

// Event sent to hammer to trigger an image compression.
type ImageCompressTriggerEvent struct {
	Identifier string // identifier
	FileId     string // file id from mini-fstore
	ReplyTo    string // event bus that will receive event about the compressed image
}

// Event replied from hammer about the compressed image.
type ImageCompressReplyEvent struct {
	Identifier string // identifier
	FileId     string // file id from mini-fstore
}

// Event sent to hammer to trigger an vidoe thumbnail generation.
type GenVideoThumbnailTriggerEvent struct {
	Identifier string // identifier
	FileId     string // file id from mini-fstore
	ReplyTo    string // event bus that will receive event about the generated video thumbnail.
}

// Event replied from hammer about the generated video thumbnail.
type GenVideoThumbnailReplyEvent struct {
	Identifier string // identifier
	FileId     string // file id from mini-fstore
}
