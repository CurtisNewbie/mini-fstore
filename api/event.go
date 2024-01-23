package api

type UnzipFileReplyEvent struct {
	ZipFileId string
	ZipEntrys []ZipEntry
	Extra     string
}

type ZipEntry struct {
	FileId string
	Md5    string
	Name   string
	Size   int64
}
