package config

import "github.com/curtisnewbie/miso/miso"

const (
	PropStorageDir                = "fstore.storage.dir"                 // where files are stored
	PropTrashDir                  = "fstore.trash.dir"                   // where files are dumped to
	PropTempDir                   = "fstore.tmp.dir"                     // temp directory
	PropPDelStrategy              = "fstore.pdelete.strategy"            // strategy used to 'physically' delete files
	PropSanitizeStorageTaskDryRun = "task.sanitize-storage-task.dry-run" // Enable dry run for SanitizeStorageTask
	PropEnableFstoreBackup        = "fstore.backup.enabled"

	PropBackupAuthSecret = "fstore.backup.secret"
)

func init() {
	miso.SetDefProp(PropEnableFstoreBackup, false)
}
