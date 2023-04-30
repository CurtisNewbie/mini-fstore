# mini-fstore

Mini File Storage Engine ( Not really :D )

## Configuration

| Property              | Description                                    | Default Value |
| ---                   | ---                                            | ---           |
| fstore.storage.dir    | Storage Directory                              | ./storage     |
| fstore.trash.dir      | Trash Directory                                | ./trash       |
| fstore.pdelete.strategy | Strategy used to 'physically' delete files, there are two types of strategies available: direct / trash. When using 'direct' strategy, files are deleted directly. When using 'trash' strategy, files are moved into the trash directory. | trash|
| goauth.report.enabled | Whether goauth resource/path report is enabled | false         |

<!-- | fstore.server.mode    | Server Mode. There are three kinds of server mode: `cluster`, `proxy`, and `node`. <br><br>In `cluster` mode, servers operate as a single cluster, if every one of them is connected to the same database and using the same disk, they will behave exactly the same. <br><br> But there will be cases where we need to deploy the servers on different machines using different disks. This is when we use `proxy` + `node` mode. The one with `proxy` mode will behave just like a proxy, and the servers with `node` mode will be responsible for storing the actual files. | cluster | -->

## API

| Method | Path | Parameters | Description |
| --- | --- | --- | --- |
| GET | /file/raw | key (QUERY) | Download file by a randomly generated file key |
| POST | /file | filename (HEADER) | Upload file |
| GET | /file/info | fileId (QUERY) | Get file's infomation by fileId |
| GET | /file/key/random | fileId (QUERY) | Generate random file key for the file |
| DELETE | /file | fileId (QUERY) | Delete file logically |