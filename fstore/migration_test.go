package fstore

import (
	"testing"

	"github.com/curtisnewbie/miso/miso"
)

func TestMigrateFileServer(t *testing.T) {
	ag := []string{"configFile=../app-conf-dev.yml"}
	c := miso.EmptyRail()
	miso.DefaultReadConfig(ag, c)
	miso.ConfigureLogging(c)
	if e := miso.InitMySQLFromProp(); e != nil {
		t.Fatal(e)
	}

	miso.SetProp(PROP_STORAGE_DIR, "../storage_test")
	miso.SetProp(PROP_TRASH_DIR, "../trash_test")
	miso.SetProp(PROP_MIGR_FILE_SERVER_DRY_RUN, true)
	miso.SetProp(PROP_MIGR_FILE_SERVER_STORAGE, "/Users/zhuangyongj/file")
	if e := MigrateFileServer(c); e != nil {
		t.Fatal(e)
	}
}
