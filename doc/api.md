# API Endpoints

- GET /file/stream
  - Description: Media streaming using temporary file key, the file_key's ttl is extended with each subsequent request. This endpoint is expected to be accessible publicly without authorization, since a temporary file_key is generated and used.
  - Expected Access Scope: PUBLIC
  - Query Parameter:
    - "key": temporary file key
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8084/file/stream?key='
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    let key: any | null = null;
    this.http.get<any>(`/fstore/file/stream?key=${key}`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /file/raw
  - Description: Download file using temporary file key. This endpoint is expected to be accessible publicly without authorization, since a temporary file_key is generated and used.
  - Expected Access Scope: PUBLIC
  - Query Parameter:
    - "key": temporary file key
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8084/file/raw?key='
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    let key: any | null = null;
    this.http.get<any>(`/fstore/file/raw?key=${key}`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- PUT /file
  - Description: Upload file. A temporary file_id is returned, which should be used to exchange the real file_id
  - Bound to Resource: `"fstore-upload"`
  - Header Parameter:
    - "filename": name of the uploaded file
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (string) response data
  - cURL:
    ```sh
    curl -X PUT 'http://localhost:8084/file' \
      -H 'filename: '
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: string                  // response data
    }
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    let filename: any | null = null;
    this.http.put<any>(`/fstore/file`,
      {
        headers: {
          "filename": filename
        }
      })
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: string = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /file/info
  - Description: Fetch file info
  - Query Parameter:
    - "fileId": actual file_id of the file record
    - "uploadFileId": temporary file_id returned when uploading files
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (FstoreFile) response data
      - "fileId": (string) file unique identifier
      - "name": (string) file name
      - "status": (string) status, 'NORMAL', 'LOG_DEL' (logically deleted), 'PHY_DEL' (physically deleted)
      - "size": (int64) file size in bytes
      - "md5": (string) MD5 checksum
      - "uplTime": (int64) upload time
      - "logDelTime": (int64) logically deleted at
      - "phyDelTime": (int64) physically deleted at
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8084/file/info?fileId=&uploadFileId='
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: FstoreFile
    }
    export interface FstoreFile {
      fileId?: string                // file unique identifier
      name?: string                  // file name
      status?: string                // status, 'NORMAL', 'LOG_DEL' (logically deleted), 'PHY_DEL' (physically deleted)
      size?: number                  // file size in bytes
      md5?: string                   // MD5 checksum
      uplTime?: number               // upload time
      logDelTime?: number            // logically deleted at
      phyDelTime?: number            // physically deleted at
    }
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    let fileId: any | null = null;
    let uploadFileId: any | null = null;
    this.http.get<any>(`/fstore/file/info?fileId=${fileId}&uploadFileId=${uploadFileId}`)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: FstoreFile = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /file/key
  - Description: Generate temporary file key for downloading and streaming. This endpoint is expected to be called internally by another backend service that validates the ownership of the file properly.
  - Query Parameter:
    - "fileId": actual file_id of the file record
    - "filename": the name that will be used when downloading the file
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (string) response data
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8084/file/key?fileId=&filename='
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: string                  // response data
    }
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    let fileId: any | null = null;
    let filename: any | null = null;
    this.http.get<any>(`/fstore/file/key?fileId=${fileId}&filename=${filename}`)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: string = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /file/direct
  - Description: Download files directly using file_id. This endpoint is expected to be protected and only used internally by another backend service. Users can eaily steal others file_id and attempt to download the file, so it's better not be exposed to the end users.
  - Query Parameter:
    - "fileId": actual file_id of the file record
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8084/file/direct?fileId='
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    let fileId: any | null = null;
    this.http.get<any>(`/fstore/file/direct?fileId=${fileId}`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- DELETE /file
  - Description: Mark file as deleted.
  - Query Parameter:
    - "fileId": actual file_id of the file record
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
  - cURL:
    ```sh
    curl -X DELETE 'http://localhost:8084/file?fileId='
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
    }
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    let fileId: any | null = null;
    this.http.delete<any>(`/fstore/file?fileId=${fileId}`)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- POST /file/unzip
  - Description: Unzip archive, upload all the zip entries, and reply the final results back to the caller asynchronously
  - JSON Request:
    - "fileId": (string) file_id of zip file
    - "replyToEventBus": (string) name of the rabbitmq exchange to reply to, routing_key is '#'
    - "extra": (string) extra information that will be passed around for the caller
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
  - cURL:
    ```sh
    curl -X POST 'http://localhost:8084/file/unzip' \
      -H 'Content-Type: application/json' \
      -d '{"extra":"","fileId":"","replyToEventBus":""}'
    ```

  - JSON Request Object In TypeScript:
    ```ts
    export interface UnzipFileReq {
      fileId?: string                // file_id of zip file
      replyToEventBus?: string       // name of the rabbitmq exchange to reply to, routing_key is '#'
      extra?: string                 // extra information that will be passed around for the caller
    }
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
    }
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    let req: UnzipFileReq | null = null;
    this.http.post<any>(`/fstore/file/unzip`, req)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- POST /backup/file/list
  - Description: Backup tool list files
  - Expected Access Scope: PUBLIC
  - Header Parameter:
    - "Authorization": Basic Authorization
  - JSON Request:
    - "limit": (int64) 
    - "idOffset": (int) 
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (ListBackupFileResp) response data
      - "files": ([]fstore.BackupFileInf) 
        - "id": (int64) 
        - "fileId": (string) 
        - "name": (string) 
        - "status": (string) 
        - "size": (int64) 
        - "md5": (string) 
  - cURL:
    ```sh
    curl -X POST 'http://localhost:8084/backup/file/list' \
      -H 'Authorization: ' \
      -H 'Content-Type: application/json' \
      -d '{"idOffset":0,"limit":0}'
    ```

  - JSON Request Object In TypeScript:
    ```ts
    export interface ListBackupFileReq {
      limit?: number
      idOffset?: number
    }
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: ListBackupFileResp
    }
    export interface ListBackupFileResp {
      files?: BackupFileInf[]
    }
    export interface BackupFileInf {
      id?: number
      fileId?: string
      name?: string
      status?: string
      size?: number
      md5?: string
    }
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    let authorization: any | null = null;
    let req: ListBackupFileReq | null = null;
    this.http.post<any>(`/fstore/backup/file/list`, req,
      {
        headers: {
          "Authorization": authorization
        }
      })
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: ListBackupFileResp = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /backup/file/raw
  - Description: Backup tool download file
  - Expected Access Scope: PUBLIC
  - Header Parameter:
    - "Authorization": Basic Authorization
  - Query Parameter:
    - "fileId": actual file_id of the file record
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8084/backup/file/raw?fileId=' \
      -H 'Authorization: '
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    let fileId: any | null = null;
    let authorization: any | null = null;
    this.http.get<any>(`/fstore/backup/file/raw?fileId=${fileId}`,
      {
        headers: {
          "Authorization": authorization
        }
      })
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- POST /maintenance/remove-deleted
  - Description: Remove files that are logically deleted and not linked (symbolically)
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
  - cURL:
    ```sh
    curl -X POST 'http://localhost:8084/maintenance/remove-deleted'
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
    }
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    this.http.post<any>(`/fstore/maintenance/remove-deleted`)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- POST /maintenance/sanitize-storage
  - Description: Sanitize storage, remove files in storage directory that don't exist in database
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
  - cURL:
    ```sh
    curl -X POST 'http://localhost:8084/maintenance/sanitize-storage'
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
    }
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    this.http.post<any>(`/fstore/maintenance/sanitize-storage`)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- POST /maintenance/compute-checksum
  - Description: Compute files' checksum if absent
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
  - cURL:
    ```sh
    curl -X POST 'http://localhost:8084/maintenance/compute-checksum'
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface Resp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
    }
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    this.http.post<any>(`/fstore/maintenance/compute-checksum`)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /auth/resource
  - Description: Expose resource and endpoint information to other backend service for authorization.
  - Expected Access Scope: PROTECTED
  - JSON Response:
    - "errorCode": (string) error code
    - "msg": (string) message
    - "error": (bool) whether the request was successful
    - "data": (ResourceInfoRes) response data
      - "resources": ([]auth.Resource) 
        - "name": (string) resource name
        - "code": (string) resource code, unique identifier
      - "paths": ([]auth.Endpoint) 
        - "type": (string) access scope type: PROTECTED/PUBLIC
        - "url": (string) endpoint url
        - "group": (string) app name
        - "desc": (string) description of the endpoint
        - "resCode": (string) resource code
        - "method": (string) http method
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8084/auth/resource'
    ```

  - JSON Response Object In TypeScript:
    ```ts
    export interface GnResp {
      errorCode?: string             // error code
      msg?: string                   // message
      error?: boolean                // whether the request was successful
      data?: ResourceInfoRes
    }
    export interface ResourceInfoRes {
      resources?: Resource[]
      paths?: Endpoint[]
    }
    export interface Resource {
      name?: string                  // resource name
      code?: string                  // resource code, unique identifier
    }
    export interface Endpoint {
      type?: string                  // access scope type: PROTECTED/PUBLIC
      url?: string                   // endpoint url
      group?: string                 // app name
      desc?: string                  // description of the endpoint
      resCode?: string               // resource code
      method?: string                // http method
    }
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    this.http.get<any>(`/fstore/auth/resource`)
      .subscribe({
        next: (resp) => {
          if (resp.error) {
            this.snackBar.open(resp.msg, "ok", { duration: 6000 })
            return;
          }
          let dat: ResourceInfoRes = resp.data;
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /metrics
  - Description: Collect prometheus metrics information
  - Header Parameter:
    - "Authorization": Basic authorization if enabled
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8084/metrics' \
      -H 'Authorization: '
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    let authorization: any | null = null;
    this.http.get<any>(`/fstore/metrics`,
      {
        headers: {
          "Authorization": authorization
        }
      })
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /debug/pprof
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8084/debug/pprof'
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    this.http.get<any>(`/fstore/debug/pprof`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /debug/pprof/:name
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8084/debug/pprof/:name'
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    this.http.get<any>(`/fstore/debug/pprof/:name`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /debug/pprof/cmdline
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8084/debug/pprof/cmdline'
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    this.http.get<any>(`/fstore/debug/pprof/cmdline`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /debug/pprof/profile
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8084/debug/pprof/profile'
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    this.http.get<any>(`/fstore/debug/pprof/profile`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /debug/pprof/symbol
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8084/debug/pprof/symbol'
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    this.http.get<any>(`/fstore/debug/pprof/symbol`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /debug/pprof/trace
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8084/debug/pprof/trace'
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    this.http.get<any>(`/fstore/debug/pprof/trace`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

- GET /doc/api
  - Description: Serve the generated API documentation webpage
  - Expected Access Scope: PUBLIC
  - cURL:
    ```sh
    curl -X GET 'http://localhost:8084/doc/api'
    ```

  - Angular HttpClient Demo:
    ```ts
    import { MatSnackBar } from "@angular/material/snack-bar";
    import { HttpClient } from "@angular/common/http";

    constructor(
      private snackBar: MatSnackBar,
      private http: HttpClient
    ) {}

    this.http.get<any>(`/fstore/doc/api`)
      .subscribe({
        next: () => {
        },
        error: (err) => {
          console.log(err)
          this.snackBar.open("Request failed, unknown error", "ok", { duration: 3000 })
        }
      });
    ```

# Event Pipelines

- GenImgThumbnailPipeline
  - Description: Pipeline to trigger async image thumbnail generation, will reply api.ImageCompressReplyEvent when the processing succeeds.
  - RabbitMQ Queue: `event.bus.fstore.image.compress.processing`
  - RabbitMQ Exchange: `event.bus.fstore.image.compress.processing`
  - RabbitMQ RoutingKey: `#`
  - Event Payload:
    - "identifier": (string) identifier
    - "fileId": (string) file id from mini-fstore
    - "replyTo": (string) event bus that will receive event about the generated image thumbnail.

- GenVidThumbnailPipeline
  - Description: Pipeline to trigger async video thumbnail generation, will reply api.GenVideoThumbnailReplyEvent when the processing succeeds.
  - RabbitMQ Queue: `event.bus.fstore.video.thumbnail.processing`
  - RabbitMQ Exchange: `event.bus.fstore.video.thumbnail.processing`
  - RabbitMQ RoutingKey: `#`
  - Event Payload:
    - "identifier": (string) dentifier
    - "fileId": (string) file id from mini-fstore
    - "replyTo": (string) event bus that will receive event about the generated video thumbnail.
