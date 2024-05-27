package server

import (
	"os"

	"github.com/curtisnewbie/mini-fstore/internal/fstore"
	"github.com/curtisnewbie/mini-fstore/internal/hammer"
	"github.com/curtisnewbie/miso/middleware/user-vault/common"
	"github.com/curtisnewbie/miso/miso"
)

func init() {
	miso.PreServerBootstrap(func(rail miso.Rail) error {
		rail.Infof("mini-fstore version: %v", Version)
		return nil
	})
}

func BootstrapServer(args []string) {
	common.LoadBuiltinPropagationKeys()
	miso.PreServerBootstrap(fstore.TryMigrateFileServer)
	miso.PreServerBootstrap(fstore.RegisterRoutes)
	miso.PreServerBootstrap(fstore.PrepareEventBus)
	miso.PreServerBootstrap(fstore.InitTrashDir)
	miso.PreServerBootstrap(fstore.InitStorageDir)
	miso.PreServerBootstrap(hammer.InitPipeline)
	miso.BootstrapServer(os.Args)
}