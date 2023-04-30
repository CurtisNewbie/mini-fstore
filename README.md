# mini-fstore

Mini File Storage Engine ( Not really :D )

## Configuration

| Property              | Description                                    | Default Value |
| ---                   | ---                                            | ---           |
| fstore.storage.dir    | Storage Directory                              | ./storage     |
| fstore.trash.dir      | Trash Directory                                | ./trash       |
| fstore.pdelete.strategy | Strategy used to 'physically' delete files, there are two types of strategies available: direct / trash. When using 'direct' strategy, files are deleted directly. When using 'trash' strategy, files are moved into the trash directory. | trash|
| goauth.report.enabled | Whether goauth resource/path report is enabled | false         |

## API

| Method | Path | Parameters | Description |
| --- | --- | --- | --- |
| GET | /file/raw | key (QUERY) | Download file by a randomly generated file key |
| POST | /file | filename (HEADER) | Upload file |
| GET | /file/info | fileId (QUERY) | Get file's infomation by fileId |
| GET | /file/key/random | fileId (QUERY) | Generate random file key for the file |
| DELETE | /file | fileId (QUERY) | Delete file logically |