package hammer

import (
	"errors"
	"fmt"
	"os"

	"github.com/curtisnewbie/mini-fstore/api"
	"github.com/curtisnewbie/mini-fstore/internal/fstore"
	"github.com/curtisnewbie/miso/miso"
)

func InitPipeline(rail miso.Rail) error {
	api.GenImgThumbnailPipeline.Listen(3, ListenCompressImageEvent)
	api.GenVidThumbnailPipeline.Listen(3, ListenGenVideoThumbnailEvent)
	return nil
}

func ListenCompressImageEvent(rail miso.Rail, evt api.ImageCompressTriggerEvent) error {
	rail.Infof("Received CompressImageEvent: %+v", evt)

	generatedFileId, err := CompressImage(rail, evt)
	if err != nil {
		return err
	}
	if evt.ReplyTo == "" {
		rail.Warn("ImageCompressTriggerEvent.ReplyTo is empty")
		return nil
	}
	// reply to the specified event bus
	return miso.PubEventBus(rail,
		api.ImageCompressReplyEvent{Identifier: evt.Identifier, FileId: generatedFileId},
		evt.ReplyTo)
}

func ListenGenVideoThumbnailEvent(rail miso.Rail, evt api.GenVideoThumbnailTriggerEvent) error {
	rail.Infof("Received GenVideoThumbnailTriggerEvent: %+v", evt)

	generatedFileId, err := GenerateVideoThumbnail(rail, evt)
	if err != nil {
		return err
	}
	if evt.ReplyTo == "" {
		rail.Warn("GenVideoThumbnailTriggerEvent.ReplyTo is empty")
		return nil
	}
	// reply to the specified event bus
	return miso.PubEventBus(rail,
		api.GenVideoThumbnailReplyEvent{Identifier: evt.Identifier, FileId: generatedFileId},
		evt.ReplyTo)
}

func CompressImage(rail miso.Rail, evt api.ImageCompressTriggerEvent) (string, error) {
	originFile, err := fstore.FindFile(miso.GetMySQL(), evt.FileId)
	if err != nil {
		if errors.Is(err, fstore.ErrFileDeleted) || errors.Is(err, fstore.ErrFileNotFound) {
			rail.Warnf("File %v is not found or deleted, %v", evt.FileId, evt.Identifier)
			return "", nil
		}
		return "", fmt.Errorf("failed to fetch fstore file info: %v, %v", evt.FileId, err)
	}

	// download the origin file from mini-fstore
	downloadPath := "/tmp/" + miso.RandNum(20)
	downloadFile, err := os.Create(downloadPath)
	if err != nil {
		return "", err
	}
	defer downloadFile.Close()

	if e := fstore.TransferWholeFile(rail, downloadFile, evt.FileId); e != nil {
		rail.Errorf("Failed to DownloadFstoreFile, %v", e)
		return "", nil
	}
	rail.Infof("File downloaded to %v", downloadPath)
	defer os.Remove(downloadPath)

	// compress the origin image, if the compression failed, we just give up
	compressPath := downloadPath + "_compressed"
	if e := GiftCompressImage(rail, downloadPath, compressPath); e != nil {
		rail.Errorf("Failed to compress image, giving up, %v", e)
		return "", nil // don't retry
	}
	defer os.Remove(compressPath)
	rail.Infof("Image %v compressed to %v", evt.Identifier, compressPath)

	// upload the compressed image to mini-fstore
	compressedFile, err := os.Open(compressPath)
	if err != nil {
		return "", fmt.Errorf("failed to open compressed file, file: %v, %v", compressPath, err)
	}
	defer compressedFile.Close()

	uploadFileId, e := fstore.UploadFile(rail, compressedFile, originFile.Name+"_thumbnail")
	if e != nil {
		rail.Errorf("Failed to UploadFstoreFile, %v", e)
		return "", fmt.Errorf("failed to upload fstore file, %v", e)
	}

	// exchange the uploadFileId with the real fileId
	thumbnailFile, e := fstore.FindFile(miso.GetMySQL(), uploadFileId)
	if e != nil {
		rail.Errorf("Failed to FetchFstoreFileInfo, %v", e)
		return "", fmt.Errorf("failed to fetch fstore file info, %v", e)
	}

	return thumbnailFile.FileId, nil
}

func GenerateVideoThumbnail(rail miso.Rail, evt api.GenVideoThumbnailTriggerEvent) (string, error) {
	rail.Infof("Received GenVideoThumbnailTriggerEvent: %+v", evt)

	originFile, err := fstore.FindFile(miso.GetMySQL(), evt.FileId)
	if err != nil {
		if errors.Is(err, fstore.ErrFileDeleted) || errors.Is(err, fstore.ErrFileNotFound) {
			rail.Warnf("File %v is not found or deleted, %v", evt.FileId, evt.Identifier)
			return "", nil
		}
		return "", fmt.Errorf("failed to fetch fstore file info: %v, %v", evt.FileId, err)
	}

	// temp path for ffmpeg to extract first frame of the video
	genPath := "/tmp/" + miso.RandNum(20) + ".png"
	defer os.Remove(genPath)

	storagePath := fstore.GenStoragePath(evt.FileId)
	if err := ExtractFirstFrame(rail, storagePath, genPath); err != nil {
		rail.Errorf("Failed to generate video thumbnail, giving up, %v", err)
		return "", nil
	}
	rail.Infof("Video (%v)'s first frame is generated to %v", evt.Identifier, genPath)

	// upload the compressed image to mini-fstore
	genFile, err := os.Open(genPath)
	if err != nil {
		return "", fmt.Errorf("failed to open generated video thumbnail file: %v, %w", genFile, err)
	}
	defer genFile.Close()

	uploadFileId, e := fstore.UploadFile(rail, genFile, originFile.Name+"_thumbnail")
	if e != nil {
		rail.Errorf("Failed to UploadFile, %v", e)
		return "", fmt.Errorf("failed to upload fstore file, %v", e)
	}

	// exchange the uploadFileId with the real fileId
	thumbnailFile, e := fstore.FindFile(miso.GetMySQL(), uploadFileId)
	if e != nil {
		rail.Errorf("Failed to FetchFileInfo, %v", e)
		return "", fmt.Errorf("failed to fetch fstore file info, %w", e)
	}

	return thumbnailFile.FileId, nil
}
