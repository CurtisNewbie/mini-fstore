package main

import (
	"os"

	"github.com/curtisnewbie/mini-fstore/internal/server"
)

func main() {
	server.BootstrapServer(os.Args)
}
