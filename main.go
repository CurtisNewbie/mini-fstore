package main

import (
	"os"

	"github.com/curtisnewbie/gocommon/server"
)

func main() {	
	server.DefaultBootstrapServer(os.Args)
}