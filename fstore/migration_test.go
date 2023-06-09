package fstore

import (
	"testing"

	"github.com/curtisnewbie/gocommon/common"
	"github.com/curtisnewbie/gocommon/mysql"
	"github.com/curtisnewbie/gocommon/server"
)

func TestMigrateFileServer(t *testing.T) {
	ag := []string{"configFile=../app-conf-dev.yml"}
	c := common.EmptyExecContext()
	common.DefaultReadConfig(ag, c)
	server.ConfigureLogging(c)
	if e := mysql.InitMySqlFromProp(); e != nil {
		t.Fatal(e)
	}

	common.SetProp(PROP_STORAGE_DIR, "../storage_test")
	common.SetProp(PROP_TRASH_DIR, "../trash_test")
	common.SetProp(PROP_MIGR_FILE_SERVER_DRY_RUN, true)
	common.SetProp(PROP_MIGR_FILE_SERVER_STORAGE, "/Users/zhuangyongj/file")
	if e := MigrateFileServer(c); e != nil {
		t.Fatal(e)
	}
}
