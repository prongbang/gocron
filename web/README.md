# gocron web

Dashboard for the gocron scheduler API. `bun run build` writes a static SPA to `build/`, which `web.go` embeds into the gocron binary (`make build` at the repo root does both).

Develop against a running gocron (`GOCRON_API=true`); `/v1` is proxied to it:

```shell
bun install
bun run dev                                   # proxies to http://localhost:8000
GOCRON_API=http://other-host:8000 bun run dev  # or somewhere else
```
