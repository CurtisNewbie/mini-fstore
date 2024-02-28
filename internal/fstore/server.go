package fstore

import (
	"os"

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
	miso.PreServerBootstrap(TryMigrateFileServer)
	miso.PreServerBootstrap(registerRoutes)
	miso.PreServerBootstrap(PrepareEventBus)
	miso.PreServerBootstrap(InitTrashDir)
	miso.PreServerBootstrap(InitStorageDir)
	miso.BootstrapServer(os.Args)
}
