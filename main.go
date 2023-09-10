package main

import (
	"os"

	"github.com/curtisnewbie/mini-fstore/fstore"
	"github.com/curtisnewbie/miso/miso"
)

func main() {
	miso.PreServerBootstrap(func(c miso.Rail) error {
		return fstore.PrepareServer(c)
	})
	miso.BootstrapServer(os.Args)
}
