package api

import "github.com/curtisnewbie/miso/miso"

var (
	// Pipeline to trigger image compression
	//
	// Reply api.ImageCompressReplyEvent when the processing succeeds.
	GenImgThumbnailPipeline = miso.NewEventPipeline[ImageCompressTriggerEvent]("event.bus.fstore.image.compress.processing")

	// Pipeline to trigger video thumbnail generation
	//
	// Reply api.GenVideoThumbnailReplyEvent when the processing succeeds.
	GenVidThumbnailPipeline = miso.NewEventPipeline[GenVideoThumbnailTriggerEvent]("event.bus.fstore.video.thumbnail.processing")
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
