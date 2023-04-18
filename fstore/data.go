package fstore

import (
	"github.com/curtisnewbie/gocommon/common"
)

type File struct {
	Id int64
	FileId string
	Status string
	Size int64
	Md5 string
	UplTime common.ETime
	DelTime common.ETime
}
