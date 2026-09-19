# gocron [![Docker Pulls](https://img.shields.io/docker/pulls/prongbang/gocron.svg)](https://hub.docker.com/r/prongbang/gocron/) [![Image Size](https://img.shields.io/docker/image-size/prongbang/gocron.svg)](https://hub.docker.com/r/prongbang/gocron/)

[Gocron](https://hub.docker.com/r/prongbang/gocron) manages cron jobs with a configuration.

[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/prongbang)

## Install by Source

```shell
go get github.com/prongbang/gocron
```

Using

```go
package main

import (
	"github.com/prongbang/gocron/cmd/gocron/runner"
	_ "time/tzdata"
)

func main() {
	runner.Start()
}
```

## Install by Docker

## Configuration with API

- docker-compose.yml

```yaml
services:
  gocron:
    image: prongbang/gocron:latest
    ports:
      - "8000:8000"
    environment:
      - TZ=Asia/Bangkok
      - GOCRON_API=true
```

### Create

- `POST http://localhost:8000/v1/scheduler`

Request

```json
{
    "project": "billing",
    "cron": "*/1 * * * *",
    "task": {
        "type": "api",
        "config": {
            "url": "http://localhost/notify",
            "method": "POST",
            "header": {
                "X-API-KEY": "ABC"
            },
            "body": {
                "data": "Hi"
            }
        }
    }
}
```

Response

```json
{
    "code": "201",
    "message": "Created",
    "data": {
        "job": "83ba2dc9dd5c4326a07dc9eb2d5163b3"
    }
}
```

### Get all

- `GET http://localhost:8000/v1/scheduler`

Response

```json
{
    "code": "200",
    "message": "OK",
    "data": [
        {
            "job": "83ba2dc9dd5c4326a07dc9eb2d5163b3",
            "project": "billing",
            "cron": "*/1 * * * *",
            "task": {
                "type": "api",
                "config": {
                    "url": "http://localhost/notify",
                    "method": "POST",
                    "body": {
                      "data": "Hi"
                    },
                    "header": {
                        "X-API-KEY": "ABC"
                    }
                }
            },
            "running": true,
            "next_run": "2026-09-19T09:41:00+07:00"
        }
    ]
}
```

### Stop

- `POST http://localhost:8000/v1/scheduler/stop`

Request

```json
{
    "job": "83ba2dc9dd5c4326a07dc9eb2d5163b3"
}
```

Response

```json
{
    "code": "200",
    "message": "OK",
    "data": {
        "job": "83ba2dc9dd5c4326a07dc9eb2d5163b3"
    }
}
```

### History

Every run of an API-created job is recorded and kept for 7 days (not recorded in BuildIn mode).

- `GET http://localhost:8000/v1/history?project=billing&status=failed&q=timeout&page=1&limit=20`

| Param | Description |
|---|---|
| `job` | Only runs of this job id |
| `project` | Only runs of this project |
| `status` | `ok` (2xx) or `failed` (anything else) |
| `q` | Case-insensitive search in job, project, method, url, status and response |
| `page` | 1-based, default `1` |
| `limit` | 1-100, default `20` |

Response (newest first; `projects` lists every project in history, for filters)

```json
{
    "code": "200",
    "message": "OK",
    "data": {
        "items": [
            {
                "job": "83ba2dc9dd5c4326a07dc9eb2d5163b3",
                "project": "billing",
                "cron": "*/1 * * * *",
                "method": "POST",
                "url": "http://localhost/notify",
                "status": 504,
                "response": "gateway timeout",
                "started_at": "2026-09-19T09:49:00.0012+07:00",
                "duration_ms": 60000
            }
        ],
        "total": 1,
        "projects": ["billing", "reports"]
    }
}
```

## Configuration with BuildIn

```yml
schedulers:
  - job: "every-24-hours"
    cron: "0 0 * * *"
    task:
      url: "http://localhost/post"
      method: "POST"
      body: >
        {"data": "every-24-hours"}
      header: >
        {"X-Api-Key": "XXX"}
```

### Config from File

- Project structure

```shell
project
├── configuration
│     └── configuration.yml
└── docker-compose.yml
```

- docker-compose.yml

```yml
services:
  gocron:
    image: prongbang/gocron:latest
    environment:
      - TZ=Asia/Bangkok
      - GOCRON_API=true
      - GOCRON_BUILDIN=true
      - GOCRON_SOURCE=file
    volumes:
      - "./configuration.yml:/app/configuration/configuration.yml"
```

### Config from Remote Key/Value Store Example - Encrypted

- docker-compose.yml

```yml
services:
  gocron:
    image: prongbang/gocron:latest
    environment:
      - TZ=Asia/Bangkok
      - GOCRON_BUILDIN=true
      - GOCRON_SOURCE=remote
      - GOCRON_REMOTE_SECURE=true
      - GOCRON_REMOTE_PROVIDER=http://127.0.0.1:4001
      - GOCRON_REMOTE_ENDPOINT=true
      - GOCRON_REMOTE_PATH=/config/hugo.yml
      - GOCRON_REMOTE_SECRET_KEYRING=/etc/secrets/mykeyring.gpg
```

### Config from Remote Key/Value Store Example - Unencrypted

- docker-compose.yml

```yml
services:
  gocron:
    image: prongbang/gocron:latest
    environment:
      - TZ=Asia/Bangkok
      - GOCRON_BUILDIN=true
      - GOCRON_SOURCE=remote
      - GOCRON_REMOTE_SECURE=true
      - GOCRON_REMOTE_PROVIDER=http://127.0.0.1:4001
      - GOCRON_REMOTE_ENDPOINT=true
      - GOCRON_REMOTE_PATH=/config/hugo.yml
```
