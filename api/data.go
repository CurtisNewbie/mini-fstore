package api

import "github.com/curtisnewbie/miso/miso"

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
