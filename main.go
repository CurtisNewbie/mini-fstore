package main

import (
	"os"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/server"
	"github.com/curtisnewbie/mini-fstore/fstore"
)

func main() {
	server.PreServerBootstrap(func(c common.Rail) error {
		fstore.PrepareServer(c)
		return nil
	})
	server.BootstrapServer(os.Args)
}
