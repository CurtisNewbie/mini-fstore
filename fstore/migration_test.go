package fstore

import (
	"testing"

	"github.com/curtisnewbie/miso/core"
	"github.com/curtisnewbie/miso/mysql"
	"github.com/curtisnewbie/miso/server"
)

func TestMigrateFileServer(t *testing.T) {
	ag := []string{"configFile=../app-conf-dev.yml"}
	c := core.EmptyRail()
	core.DefaultReadConfig(ag, c)
	server.ConfigureLogging(c)
	if e := mysql.InitMySqlFromProp(); e != nil {
		t.Fatal(e)
	}

	core.SetProp(PROP_STORAGE_DIR, "../storage_test")
	core.SetProp(PROP_TRASH_DIR, "../trash_test")
	core.SetProp(PROP_MIGR_FILE_SERVER_DRY_RUN, true)
	core.SetProp(PROP_MIGR_FILE_SERVER_STORAGE, "/Users/zhuangyongj/file")
	if e := MigrateFileServer(c); e != nil {
		t.Fatal(e)
	}
}
