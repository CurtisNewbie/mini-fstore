package main

import (
	"os"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/server"
	"github.com/curtisnewbie/mini-fstore/fstore"
)

func main() {
	c := common.EmptyExecContext()
	server.DefaultBootstrapServer(os.Args, c, func() error {
		fstore.PrepareServer(c)
		return nil
	})
}
