package fstore

const (
	/*
		--------------------------------------------------

		Mini-Fstore

		--------------------------------------------------
	*/

	PROP_STORAGE_DIR   = "fstore.storage.dir"      // where files are stored
	PROP_TRASH_DIR     = "fstore.trash.dir"        // where files are dumped to
	PROP_PDEL_STRATEGY = "fstore.pdelete.strategy" // strategy used to 'physically' delete files
	PROP_SERVER_MODE   = "fstore.server.mode"      // server mode the fstore is in
	PROP_SERVER_LIST   = "fstore.server.list"      // fstore server list

	/*
		--------------------------------------------------

		For Migration

		--------------------------------------------------
	*/

	PROP_MIGR_FILE_SERVER_ENABLED        = "fstore.migr.file-server.enabled"        // file-server migration enabled
	PROP_MIGR_FILE_SERVER_DRY_RUN        = "fstore.migr.file-server.dry-run"        // dry run file-server migration
	PROP_MIGR_FILE_SERVER_STORAGE        = "fstore.migr.file-server.storage"        // location file-server's files, files are copied to mini-fstore's server manually
	PROP_MIGR_FILE_SERVER_MYSQL_USER     = "fstore.migr.file-server.mysql.user"     // file-server's db mysql user
	PROP_MIGR_FILE_SERVER_MYSQL_PWD      = "fstore.migr.file-server.mysql.password" // file-server's db mysql password
	PROP_MIGR_FILE_SERVER_MYSQL_DATABASE = "fstore.migr.file-server.mysql.database" // file-server's db mysql schema name
	PROP_MIGR_FILE_SERVER_MYSQL_HOST     = "fstore.migr.file-server.mysql.host"     // file-server's db mysql host
	PROP_MIGR_FILE_SERVER_MYSQL_PORT     = "fstore.migr.file-server.mysql.port"     // file-server's db mysql port

	/*
		--------------------------------------------------

		For GoAuth

		--------------------------------------------------
	*/

	PROP_ENABLE_GOAUTH_REPORT = "goauth.report.enabled" // whether goauth resource/path report is enabled
)
