package fstore

import (
	"time"

	"github.com/curtisnewbie/miso/miso"
)

var (
	UnzipResultCache = miso.NewRCache[UnzipFileReplyEvent]("mini-fstore:file:unzip:result",
		miso.RCacheConfig{
			Exp: time.Minute * 15,
		})
	UnzipPipeline = miso.NewEventPipeline[UnzipFileEvent]("mini-fstore.unzip.pipeline")
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

func PrepareEventBus(rail miso.Rail) error {
	UnzipPipeline.Listen(1, OnUnzipFileEvent)
	return nil
}

type UnzipFileEvent struct {
	FileId          string `valid:"notEmpty"`
	ReplyToEventBus string `valid:"notEmpty"`
	Extra           string
}

func OnUnzipFileEvent(rail miso.Rail, evt UnzipFileEvent) error {
	replyEvent, err := UnzipResultCache.Get(rail, evt.FileId, func() (UnzipFileReplyEvent, error) {
		entries, er := UnzipFile(rail, miso.GetMySQL(), evt)
		if er != nil {
			return UnzipFileReplyEvent{}, er
		}
		apiEntries := make([]ZipEntry, 0, len(entries))
		for _, en := range entries {
			apiEntries = append(apiEntries, ZipEntry{
				FileId: en.FileId,
				Md5:    en.Md5,
				Name:   en.Name,
				Size:   en.Size,
			})
		}
		replyEvent := UnzipFileReplyEvent{
			ZipFileId:  evt.FileId,
			ZipEntries: apiEntries,
			Extra:      evt.Extra,
		}
		return replyEvent, nil
	})

	if err != nil {
		return err
	}

	replyEvent.Extra = evt.Extra
	if err := miso.PubEventBus(rail, replyEvent, evt.ReplyToEventBus); err != nil {
		return err
	}

	return nil
}
