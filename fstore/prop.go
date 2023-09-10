package fstore

const (
	/*
		--------------------------------------------------

		Mini-Fstore

		--------------------------------------------------
	*/

	PROP_STORAGE_DIR                   = "fstore.storage.dir"                 // where files are stored
	PROP_TRASH_DIR                     = "fstore.trash.dir"                   // where files are dumped to
	PROP_PDEL_STRATEGY                 = "fstore.pdelete.strategy"            // strategy used to 'physically' delete files
	PROP_SERVER_MODE                   = "fstore.miso.mode"                 // server mode the fstore is in
	PROP_SERVER_LIST                   = "fstore.miso.list"                 // fstore server list
	PROP_SANITIZE_STORAGE_TASK_DRY_RUN = "task.sanitize-storage-task.dry-run" // Enable dry run for SanitizeStorageTask

	/*
		--------------------------------------------------

		For Migration

		--------------------------------------------------
	*/

	PROP_MIGR_FILE_SERVER_ENABLED        = "fstore.migr.file-miso.enabled"        // file-server migration enabled
	PROP_MIGR_FILE_SERVER_DRY_RUN        = "fstore.migr.file-miso.dry-run"        // dry run file-server migration
	PROP_MIGR_FILE_SERVER_STORAGE        = "fstore.migr.file-miso.storage"        // location file-server's files, files are copied to mini-fstore's server manually
	PROP_MIGR_FILE_SERVER_MYSQL_USER     = "fstore.migr.file-miso.miso.user"     // file-server's db mysql user
	PROP_MIGR_FILE_SERVER_MYSQL_PWD      = "fstore.migr.file-miso.miso.password" // file-server's db mysql password
	PROP_MIGR_FILE_SERVER_MYSQL_DATABASE = "fstore.migr.file-miso.miso.database" // file-server's db mysql schema name
	PROP_MIGR_FILE_SERVER_MYSQL_HOST     = "fstore.migr.file-miso.miso.host"     // file-server's db mysql host
	PROP_MIGR_FILE_SERVER_MYSQL_PORT     = "fstore.migr.file-miso.miso.port"     // file-server's db mysql port

	/*
		--------------------------------------------------

		For GoAuth

		--------------------------------------------------
	*/

	PROP_ENABLE_GOAUTH_REPORT = "goauth.report.enabled" // whether goauth resource/path report is enabled
)
