# mini-fstore

A mini file storage service. mini-fstore internally uses [miso](https://github.com/curtisnewbie/miso).

## Requirements

- MySQL
- Redis
- Consul
- RabbitMQ

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

Currently, mini-fstore nodes must all share the same database and the same storage devices. Some sort of distributed file system can be used and shared among all mini-fstore nodes if necessary. 

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

# API Endpoints

- GET /file/stream
  - Description: Fstore media streaming
  - Access Scope: PUBLIC
  - Query Parameter: "key"
    - Description: temporary file key
- GET /file/raw
  - Description: Fstore raw file download
  - Access Scope: PUBLIC
  - Query Parameter: "key"
    - Description: temporary file key
- PUT /file
  - Description: Fstore file upload. A temporary file_id is returned, which should be used to exchange the real file_id
  - Resource: "fstore-upload"
  - Header Parameter: "filename"
    - Description: name of the uploaded file
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (string) response data
- GET /file/info
  - Description: Fetch file info
  - Query Parameter: "uploadFileId"
    - Description: temporary file_id returned when uploading files
  - Query Parameter: "fileId"
    - Description: actual file_id of the file record
  - JSON Response:
    - "id": (int64)
    - "fileId": (string)
    - "name": (string)
    - "status": (string)
    - "size": (int64)
    - "md5": (string)
    - "uplTime": (ETime)
      - "wall": (uint64)
      - "ext": (int64)
      - "loc": (*time.Location)
    - "logDelTime": (*miso.ETime)
    - "phyDelTime": (*miso.ETime)
- GET /file/key
  - Description: Generate temporary file key for downloading and streaming
  - Query Parameter: "fileId"
    - Description: actual file_id of the file record
  - Query Parameter: "filename"
    - Description: the name that will be used when downloading the file
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (string) response data
- DELETE /file
  - Description: Make file as deleted
  - Query Parameter: "fileId"
    - Description: actual file_id of the file record
- POST /file/unzip
  - Description: Unzip archive, upload all the zip entries, and reply the final results back to the caller asynchronously
  - JSON Request:
    - "fileId": (string) file_id of zip file
    - "replyToEventBus": (string) name of the rabbitmq exchange to reply to, routing_key will always be '#'
    - "extra": (string) extra information that will be passed around for the caller
- POST /backup/file/list
  - Description: Backup tool list files
  - Access Scope: PUBLIC
  - Header Parameter: "Authorization"
    - Description: Basic Authorization
  - JSON Request:
    - "limit": (int64)
    - "idOffset": (int)
- GET /backup/file/raw
  - Description: Backup tool download file
  - Access Scope: PUBLIC
  - Header Parameter: "Authorization"
    - Description: Basic Authorization
  - Query Parameter: "fileId"
    - Description: actual file_id of the file record
- POST /maintenance/remove-deleted
  - Description: Remove files that are logically deleted and not linked (symbolically)