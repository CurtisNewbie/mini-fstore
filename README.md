# mini-fstore

Mini File Storage Engine ( Not really :D )

*Don't run the `deploy` shell script, it's for my dev setup only*

## Requirements

- MySQL
- Redis
- Consul
- [Goauth](https://github.com/CurtisNewbie/goauth)

## Configuration

| Property                           | Description                                                                                                                                                                                                                               | Default Value |
|------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------|
| fstore.storage.dir                 | Storage Directory                                                                                                                                                                                                                         | ./storage     |
| fstore.trash.dir                   | Trash Directory                                                                                                                                                                                                                           | ./trash       |
| fstore.pdelete.strategy            | Strategy used to 'physically' delete files, there are two types of strategies available: direct / trash. When using 'direct' strategy, files are deleted directly. When using 'trash' strategy, files are moved into the trash directory. | trash         |
| goauth.report.enabled              | Whether goauth resource/path report is enabled                                                                                                                                                                                            | false         |
| task.sanitize-storage-task.dry-run | Enable dry-run mode for StanitizeStorageTask                                                                                                                                                                                              | false         |

<!-- | fstore.server.mode    | Server Mode. There are three kinds of server mode: `cluster`, `proxy`, and `node`. <br><br>In `cluster` mode, servers operate as a single cluster, if every one of them is connected to the same database and using the same disk, they will behave exactly the same. <br><br> But there will be cases where we need to deploy the servers on different machines using different disks. This is when we use `proxy` + `node` mode. The one with `proxy` mode will behave just like a proxy, and the servers with `node` mode will be responsible for storing the actual files. | cluster | -->

## API

| Method | Path            | Parameters                                                                         | Description                                                                                                                                              |
|--------|-----------------|------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------|
| GET    | /file/raw       | key (QUERY)                                                                        | Download file by a randomly generated file key                                                                                                           |
| GET    | /file/stream    | key (QUERY)                                                                        | Stream file by a randomly generated file key; Byte range request is supported, the response header will always include `'Content-Type: video/mp4'`       |
| PUT    | /file           | filename (HEADER)                                                                  | Upload file, a randomly generated fake fileId is returned, can be later used to exchange the real fileId using `GET /file/info?uploadFileId=xxx` request |
| GET    | /file/info      | fileId (QUERY)<br>uploadFileId (QUERY)                                             | Get file's infomation by fileId, either use `fileId` or `uploadFileId`                                                                                   |
| GET    | /file/key       | fileId (QUERY)<br>filename (QUERY: filename used for downloading, optinoal)        | Generate random file key for the file.                                                                                                                   |
| POST   | /file/key/batch | `{"items" : [{ "fileId": "fstore file id", "filename" : "filename used for downloading" }]}` | Generate random file key for files in batch                                                                                                              |
| DELETE | /file           | fileId (QUERY)                                                                     | Delete file logically                                                                                                                                    |

## Media Streming

The `/file/stream` endpoint can be used for media streaming.

```html
<body>
    <video controls>
        <source src="http://localhost:8084/file/stream?key=0fR1H1O0t8xQZjPzbGz4lRx%2FbPacIg" type="video/mp4">
        Yo something is wrong
    </video>
</body>
```

## Migration From File-Server

Migration only supports [File-Server V1.2.7](https://github.com/CurtisNewbie/file-server). After the migration, the `File-Server` application should be replaced with [vfm](https://github.com/CurtisNewbie/vfm).

#### How Migration Works

Before the migration, files uploaded to `File-Server` must be moved/copied to mini-fstore's machine manually, and the location of these files (the base folder) must be configured correctly in `fstore.migr.file-server.storage`.

Let's assume that the database `File-Server` connects to named `fileserver`, and the database `Mini-Fstore` connects to named `mini_fstore`.

When migration is enabled, mini-fstore attempts to connect to file-server's MySQL database instance. It checks whether the table `fileserver.file_info` has the column `fstore_file_id`. If not, an error is thrown, warning that the table `fileserver.file_info` is not compatible and it cannot be migrated.

If the validation is passed, `Mini-Fstore` will read the table `fileserver.file_info` to list all the files that are not deleted and do not have any `file_id` created in column `fstore_file_id`. These files are migrated one by one, by copying files from the `fstore.migr.file-server.storage` to `fstore.storage.dir`. A new `file_id` is generated by `Mini-Fstore` and updated back to `fileserver.file_info`. The generated `file_id` can be found in `mini_fstore.file` as well.

### Configuration for File-Server Migration

| Property                               | Description                                                                                                                         | Default Value |
|----------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------|---------------|
| fstore.migr.file-server.enabled        | Enable file-server migration                                                                                                        | false         |
| fstore.migr.file-server.dry-run        | Enable file-server migration dry-run mode, files are not copied, nothing really happens, it's used mainly for debugging and testing | true          |
| fstore.migr.file-server.storage        | Where the file-server's files are located. These files must be moved to mini-fstore's machine manually before the migration.        |               |
| fstore.migr.file-server.mysql.user     | file-server's database connection username                                                                                          |               |
| fstore.migr.file-server.mysql.password | file-server's database connection password                                                                                          |               |
| fstore.migr.file-server.mysql.database | file-server's database connection schema name                                                                                       |               |
| fstore.migr.file-server.mysql.host     | file-server's database connection host                                                                                              |               |
| fstore.migr.file-server.mysql.port     | file-server's database connection port                                                                                              |               |


