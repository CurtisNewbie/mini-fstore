package main

import (
	"os"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/server"
	"github.com/curtisnewbie/mini-fstore/fstore"
)

func main() {
	common.DefaultReadConfig(os.Args)
	server.ConfigureLogging()
	fstore.PrepareWebServer()
	server.BootstrapServer()
}
