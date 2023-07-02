package main

import (
	"os"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/server"
	"github.com/curtisnewbie/mini-fstore/fstore"
)

func main() {
	server.PreServerBootstrap(func(c common.ExecContext) error {
		fstore.PrepareServer(c)
		return nil
	})
	server.BootstrapServer(os.Args)
}
