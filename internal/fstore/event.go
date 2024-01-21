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
}

func OnUnzipFileEvent(rail miso.Rail, evt UnzipFileEvent) error {
	fileIds, err := UnzipFile(rail, miso.GetMySQL(), evt)
	if err != nil {
		return err
	}
	return miso.PubEventBus(rail, api.UnzipFileReplyEvent{
		ZipFileId:       evt.FileId,
		ZipEntryFileIds: fileIds,
	}, evt.ReplyToEventBus)
}
