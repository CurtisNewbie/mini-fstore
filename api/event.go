package api

type UnzipFileReplyEvent struct {
	ZipFileId       string
	ZipEntryFileIds []string
}
