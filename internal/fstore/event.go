package fstore

import (
	"time"

	"github.com/curtisnewbie/mini-fstore/api"
	"github.com/curtisnewbie/miso/middleware/mysql"
	"github.com/curtisnewbie/miso/middleware/rabbit"
	"github.com/curtisnewbie/miso/middleware/redis"
	"github.com/curtisnewbie/miso/miso"
)

var (
	UnzipResultCache = redis.NewRCache[api.UnzipFileReplyEvent]("mini-fstore:file:unzip:result",
		redis.RCacheConfig{
			Exp: time.Minute * 15,
		})

	UnzipPipeline = rabbit.NewEventPipeline[UnzipFileEvent]("mini-fstore.unzip.pipeline").
			LogPayload().
			MaxRetry(3)
)

func InitPipeline(rail miso.Rail) error {
	UnzipPipeline.Listen(1, OnUnzipFileEvent)
	return nil
}

type UnzipFileEvent struct {
	FileId          string `valid:"notEmpty"`
	ReplyToEventBus string `valid:"notEmpty"`
	Extra           string
}

func OnUnzipFileEvent(rail miso.Rail, evt UnzipFileEvent) error {
	replyEvent, err := UnzipResultCache.Get(rail, evt.FileId, func() (api.UnzipFileReplyEvent, error) {
		entries, er := UnzipFile(rail, mysql.GetMySQL(), evt)
		if er != nil {
			return api.UnzipFileReplyEvent{}, er
		}
		apiEntries := make([]api.ZipEntry, 0, len(entries))
		for _, en := range entries {
			apiEntries = append(apiEntries, api.ZipEntry{
				FileId: en.FileId,
				Md5:    en.Md5,
				Name:   en.Name,
				Size:   en.Size,
			})
		}
		replyEvent := api.UnzipFileReplyEvent{
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
	if err := rabbit.PubEventBus(rail, replyEvent, evt.ReplyToEventBus); err != nil {
		return err
	}

	return nil
}
