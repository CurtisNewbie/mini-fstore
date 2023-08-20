package main

import (
	"os"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/server"
	"github.com/curtisnewbie/mini-fstore/fstore"
)

func main() {
	server.PreServerBootstrap(func(c common.Rail) error {
		return fstore.PrepareServer(c)
	})
	server.BootstrapServer(os.Args)
}
