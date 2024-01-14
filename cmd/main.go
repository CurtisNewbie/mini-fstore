package main

import (
	"os"

	"github.com/curtisnewbie/mini-fstore/internal/fstore"
)

func main() {
	fstore.BootstrapServer(os.Args)
}
