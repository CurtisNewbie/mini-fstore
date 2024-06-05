package api

import (
	"github.com/curtisnewbie/miso/middleware/rabbit"
)

var (
	// Pipeline to trigger async image thumbnail generation.
	//
	// Reply api.ImageCompressReplyEvent when the processing succeeds.
	GenImgThumbnailPipeline = rabbit.NewEventPipeline[ImgThumbnailTriggerEvent]("event.bus.fstore.image.compress.processing").
				LogPayload().
				MaxRetry(10).
				Document("GenImgThumbnailPipeline", "Pipeline to trigger async image thumbnail generation, will reply api.ImageCompressReplyEvent when the processing succeeds.", "fstore")

	// Pipeline to trigger async video thumbnail generation.
	//
	// Reply api.GenVideoThumbnailReplyEvent when the processing succeeds.
	GenVidThumbnailPipeline = rabbit.NewEventPipeline[VidThumbnailTriggerEvent]("event.bus.fstore.video.thumbnail.processing").
				LogPayload().
				MaxRetry(10).
				Document("GenVidThumbnailPipeline", "Pipeline to trigger async video thumbnail generation, will reply api.GenVideoThumbnailReplyEvent when the processing succeeds.", "fstore")
)

// Event sent to hammer to trigger an vidoe thumbnail generation.
type VidThumbnailTriggerEvent struct {
	Identifier string `desc:"dentifier"`
	FileId     string `desc:"file id from mini-fstore"`
	ReplyTo    string `desc:"event bus that will receive event about the generated video thumbnail."`
}

// Event sent to hammer to trigger an image compression.
type ImgThumbnailTriggerEvent struct {
	Identifier string `desc:"identifier"`
	FileId     string `desc:"file id from mini-fstore"`
	ReplyTo    string `desc:"event bus that will receive event about the generated image thumbnail."`
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
