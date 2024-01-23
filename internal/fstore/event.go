package fstore

import (
	"github.com/curtisnewbie/mini-fstore/api"
	"github.com/curtisnewbie/miso/miso"
)

const (
	UnzipPipelineEventBus = "mini-fstore.unzip.pipeline"
)

func PrepareEventBus(rail miso.Rail) error {
	miso.SubEventBus(UnzipPipelineEventBus, 1, OnUnzipFileEvent)
	return nil
}

type UnzipFileEvent struct {
	FileId          string `valid:"notEmpty"`
	ReplyToEventBus string `valid:"notEmpty"`
	Extra           string
}

func OnUnzipFileEvent(rail miso.Rail, evt UnzipFileEvent) error {
	entries, err := UnzipFile(rail, miso.GetMySQL(), evt)
	if err != nil {
		return err
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

	return miso.PubEventBus(rail, api.UnzipFileReplyEvent{
		ZipFileId:  evt.FileId,
		ZipEntries: apiEntries,
		Extra:      evt.Extra,
	}, evt.ReplyToEventBus)
}
