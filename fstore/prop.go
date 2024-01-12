package fstore

const (
	/*
		--------------------------------------------------

		Mini-Fstore

		--------------------------------------------------
	*/

	PropStorageDir                = "fstore.storage.dir"                 // where files are stored
	PropTrashDir                  = "fstore.trash.dir"                   // where files are dumped to
	PropPDelStrategy              = "fstore.pdelete.strategy"            // strategy used to 'physically' delete files
	PropServerMode                = "fstore.miso.mode"                   // server mode the fstore is in
	PropServerList                = "fstore.miso.list"                   // fstore server list
	PropSanitizeStorageTaskDryRun = "task.sanitize-storage-task.dry-run" // Enable dry run for SanitizeStorageTask

	/*
		--------------------------------------------------

		For Migration

		--------------------------------------------------
	*/

	PropMigrFileServerEnabled       = "fstore.migr.file-miso.enabled"       // file-server migration enabled
	PropMigrFileServerDryRun        = "fstore.migr.file-miso.dry-run"       // dry run file-server migration
	PropMigrFileServerStorage       = "fstore.migr.file-miso.storage"       // location file-server's files, files are copied to mini-fstore's server manually
	PropMigrFileServerMySQLUser     = "fstore.migr.file-miso.miso.user"     // file-server's db mysql user
	PropMigrFileServerMySQLPwd      = "fstore.migr.file-miso.miso.password" // file-server's db mysql password
	PropMigrFileServerMySQLDatabase = "fstore.migr.file-miso.miso.database" // file-server's db mysql schema name
	PropMigrFileServerMySQLHost     = "fstore.migr.file-miso.miso.host"     // file-server's db mysql host
	PropMigrFileServerMySQLPort     = "fstore.migr.file-miso.miso.port"     // file-server's db mysql port
)
