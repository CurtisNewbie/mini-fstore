# mini-fstore

A mini file storage service. mini-fstore internally uses [miso](https://github.com/curtisnewbie/miso).

## Requirements

- MySQL
- Redis
- Consul
- RabbitMQ
- [GoAuth](https://github.com/CurtisNewbie/goauth)(optional)

## Configuration

For more configuration, check [miso](https://github.com/curtisnewbie/miso) and [gocommon](https://github.com/CurtisNewbie/gocommon).

| Property                           | Description                                                                                                                                                                                                                               | Default Value |
|------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------|
| fstore.storage.dir                 | Storage Directory                                                                                                                                                                                                                         | ./storage     |
| fstore.trash.dir                   | Trash Directory                                                                                                                                                                                                                           | ./trash       |
| fstore.pdelete.strategy            | Strategy used to 'physically' delete files, there are two types of strategies available: direct / trash. When using 'direct' strategy, files are deleted directly. When using 'trash' strategy, files are moved into the trash directory. | trash         |
| fstore.backup.enabled              | Enable endpoints for mini-fstore file backup, see [fstore_backup](https://github.com/curtisnewbie/fstore_backup).                                                                                                                         | false         |
| fstore.backup.secret               | Secret for backup endpoints authorization, see [fstore_backup](https://github.com/curtisnewbie/fstore_backup).                                                                                                                            |               |
| task.sanitize-storage-task.dry-run | Enable dry-run mode for StanitizeStorageTask                                                                                                                                                                                              | false         |

## API

| Method | Path         | Parameters                                                                  | Description                                                                                                                                              |
|--------|--------------|-----------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------|
| GET    | /file/raw    | key (QUERY)                                                                 | Download file by a randomly generated file key                                                                                                           |
| GET    | /file/stream | key (QUERY)                                                                 | Stream file by a randomly generated file key; Byte range request is supported, the response header will always include `'Content-Type: video/mp4'`       |
| PUT    | /file        | filename (HEADER)                                                           | Upload file, a randomly generated fake fileId is returned, can be later used to exchange the real fileId using `GET /file/info?uploadFileId=xxx` request |
| GET    | /file/info   | fileId (QUERY)<br>uploadFileId (QUERY)                                      | Get file's infomation by fileId, either use `fileId` or `uploadFileId`                                                                                   |
| GET    | /file/key    | fileId (QUERY)<br>filename (QUERY: filename used for downloading, optinoal) | Generate random file key for the file.                                                                                                                   |
| DELETE | /file        | fileId (QUERY)                                                              | Delete file logically                                                                                                                                    |

## Prometheus Metrics

- `mini_fstore_generate_file_key_duration`: histogram, used to monitor the duration of each random file key generation.

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

## Limitation

Currently, mini-fstore nodes must all share the same database and the same storage advice.

## Docs

- [Migrate from file-server](./doc/file_server_migration.md)
- [Workflows](./doc/workflow.md)

## Tools

- File Backup Tools: [fstore_backup](https://github.com/CurtisNewbie/fstore_backup).

## Maintenance

mini-fstore automatically detects duplicate files by comparing filename, size and md5 checksum. If duplicate file is detected, these files are *symbolically* linked to the same file previously uploaded. This can massively reduce file storage, but multiple file records (multiple file_ids) can all point to a single file. 

Whenever a file is marked logically deleted, the file is not truely deleted. In order to cleanup the storage for the deleted files including those that are possibly symbolically linked, you have to prevent any file upload, and use the following endpoint to trigger the maintenance process:

```sh
curl -X POST http://localhost:8084/maintenance/remove-deleted
```