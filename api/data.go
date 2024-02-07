package api

import "github.com/curtisnewbie/miso/miso"

type FetchFileInfoReq struct {
	FileId       string
	UploadFileId string
}

type FstoreFile struct {
	Id         int64       `json:"id"`
	FileId     string      `json:"fileId"`
	Name       string      `json:"name"`
	Status     string      `json:"status"`
	Size       int64       `json:"size"`
	Md5        string      `json:"md5"`
	UplTime    miso.ETime  `json:"uplTime"`
	LogDelTime *miso.ETime `json:"logDelTime"`
	PhyDelTime *miso.ETime `json:"phyDelTime"`
}

type UnzipFileReq struct {
	// zip file's mini-fstore file_id.
	FileId string `valid:"notEmpty" doc:"file_id of zip file"`

	// rabbitmq exchange (both the exchange and queue must all use the same name, and are bound together using routing key '#').
	ReplyToEventBus string `valid:"notEmpty" doc:"name of the rabbitmq exchange to reply to, routing_key will always be '#'"`

	// Extra information that will be passed back to the caller in reply event.
	Extra string `doc:"extra information that will be passed around for the caller"`
}
