package fstore

import (
	"testing"

	"github.com/curtisnewbie/miso/miso"
)

func TestMigrateFileServer(t *testing.T) {
	ag := []string{"configFile=../conf.yml"}
	c := miso.EmptyRail()
	miso.DefaultReadConfig(ag, c)
	miso.ConfigureLogging(c)
	if e := miso.InitMySQLFromProp(c); e != nil {
		t.Fatal(e)
	}

	miso.SetProp(PropStorageDir, "../storage_test")
	miso.SetProp(PropTrashDir, "../trash_test")
	miso.SetProp(PropMigrFileServerDryRun, true)
	miso.SetProp(PropMigrFileServerStorage, "/Users/zhuangyongj/file")
	if e := MigrateFileServer(c); e != nil {
		t.Fatal(e)
	}
}
