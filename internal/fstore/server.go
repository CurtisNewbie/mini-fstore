package fstore

import (
	"os"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/miso/miso"
)

func TryMigrateFileServer(rail miso.Rail) error {
	if !miso.GetPropBool(PropMigrFileServerEnabled) {
		return nil
	}
	return MigrateFileServer(rail)
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
