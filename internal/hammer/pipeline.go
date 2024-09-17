package hammer

import (
	"fmt"
	"os"

	"github.com/curtisnewbie/mini-fstore/api"
	"github.com/curtisnewbie/mini-fstore/internal/fstore"
	"github.com/curtisnewbie/miso/middleware/mysql"
	"github.com/curtisnewbie/miso/middleware/rabbit"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util"
)

func InitPipeline(rail miso.Rail) error {
	api.GenImgThumbnailPipeline.Listen(3, ListenCompressImageEvent)
	api.GenVidThumbnailPipeline.Listen(3, ListenGenVideoThumbnailEvent)
	return nil
}

func ListenCompressImageEvent(rail miso.Rail, evt api.ImgThumbnailTriggerEvent) error {
	rail.Infof("Received: %#v", evt)

	if evt.ReplyTo == "" {
		rail.Warnf("replyTo is empty, %#v", evt)
		return nil
	}

	generatedFileId, err := GenImageThumbnail(rail, evt)
	if err != nil {
		return err
	}

	// reply to the specified event bus
	return rabbit.PubEventBus(rail,
		api.ImageCompressReplyEvent{Identifier: evt.Identifier, FileId: generatedFileId},
		evt.ReplyTo)
}

func ListenGenVideoThumbnailEvent(rail miso.Rail, evt api.VidThumbnailTriggerEvent) error {
	rail.Infof("Received %#v", evt)

	if evt.ReplyTo == "" {
		rail.Errorf("replyTo is empty, %#v", evt)
		return nil
	}

	generatedFileId, err := GenVideoThumbnail(rail, evt)
	if err != nil {
		return err
	}

	// reply to the specified event bus
	return rabbit.PubEventBus(rail,
		api.GenVideoThumbnailReplyEvent{Identifier: evt.Identifier, FileId: generatedFileId},
		evt.ReplyTo)
}

func GenImageThumbnail(rail miso.Rail, evt api.ImgThumbnailTriggerEvent) (string, error) {
	origin, err := fstore.FindFile(mysql.GetMySQL(), evt.FileId)
	if err != nil {
		return "", fmt.Errorf("failed to find fstore file info: %v, %v", evt.FileId, err)
	}

	if origin.Id < 1 || origin.IsDeleted() {
		rail.Warnf("fstore file %v is not found or deleted, %v", evt.FileId, evt.Identifier)
		return "", nil
	}

	// compress the origin image, if the compression failed, we just give up
	tmpPath := "/tmp/" + util.RandNum(20) + "_compressed"
	defer os.Remove(tmpPath)

	stoPath := origin.StoragePath()
	if err := GiftCompressImage(rail, stoPath, tmpPath); err != nil {
		rail.Errorf("Failed to generate image thumbnail, giving up, fileid: %v, path: %v, %v", evt.FileId, stoPath, err)
		return "", nil
	}

	rail.Infof("Image %v compressed to %v", evt.Identifier, tmpPath)

	// upload the compressed image to mini-fstore
	tmpName := origin.Name + "_thumbnail"
	uploadFileId, err := fstore.UploadLocalFile(rail, tmpPath, tmpName)
	if err != nil {
		return "", fmt.Errorf("failed to upload local fstore file, %v", err)
	}

	return uploadFileId, nil
}

func GenVideoThumbnail(rail miso.Rail, evt api.VidThumbnailTriggerEvent) (string, error) {
	origin, err := fstore.FindFile(mysql.GetMySQL(), evt.FileId)
	if err != nil {
		return "", fmt.Errorf("failed to find fstore file info: %v, %v", evt.FileId, err)
	}

	if origin.Id < 1 || origin.IsDeleted() {
		rail.Warnf("fstore file %v is not found or deleted, %v", evt.FileId, evt.Identifier)
		return "", nil
	}

	// temp path for ffmpeg to extract first frame of the video
	tmpPath := "/tmp/" + util.RandNum(20) + ".png"
	defer os.Remove(tmpPath)

	stoPath := origin.StoragePath()
	if err := ExtractFirstFrame(rail, stoPath, tmpPath); err != nil {
		rail.Errorf("Failed to generate video thumbnail, giving up, fileId: %v, path: %v, %v", evt.FileId, stoPath, err)
		return "", nil
	}

	rail.Infof("Video %v first frame is extracted to %v", evt.Identifier, tmpPath)

	// upload the extracted frame to mini-fstore
	tmpName := origin.Name + "_thumbnail"
	uploadFileId, err := fstore.UploadLocalFile(rail, tmpPath, tmpName)
	if err != nil {
		return "", fmt.Errorf("failed to upload local fstore file, %v", err)
	}

	return uploadFileId, nil
}
