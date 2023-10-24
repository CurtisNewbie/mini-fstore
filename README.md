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

## Docs

- [Migrate from file-server](./doc/file_server_migration.md)
- [Workflows](./doc/workflow.md)

## Tools

- File Backup Tools: [fstore_backup](https://github.com/CurtisNewbie/fstore_backup).