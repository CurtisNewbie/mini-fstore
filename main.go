package main

import (
	"os"

	"github.com/curtisnewbie/mini-fstore/fstore"
	"github.com/curtisnewbie/miso/core"
	"github.com/curtisnewbie/miso/server"
)

func main() {
	server.PreServerBootstrap(func(c core.Rail) error {
		return fstore.PrepareServer(c)
	})
	server.BootstrapServer(os.Args)
}
