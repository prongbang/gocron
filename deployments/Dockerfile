# web stage: build the dashboard that gets embedded into the binary
FROM oven/bun:1-alpine AS web
WORKDIR /web
COPY web/package.json web/bun.lockb ./
RUN bun install --frozen-lockfile
COPY web/ .
RUN bun run build

# build stage
FROM golang:alpine AS builder

ENV GO111MODULE=on

# install git.
RUN apk update && apk add --no-cache git

RUN mkdir -p /go/src/github.com/prongbang/gocron
WORKDIR /go/src/github.com/prongbang/gocron
COPY . .
COPY --from=web /web/build web/build

# Using go mod with go 1.11
RUN go mod vendor

# With go ≥ 1.10
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o /go/bin/gocron cmd/gocron/main.go

# small image
FROM alpine:3.18.3

WORKDIR /app
COPY --from=builder /go/src/github.com/prongbang/gocron/configuration/configuration.yml /app/configuration/configuration.yml
COPY --from=builder /go/bin/gocron .

# run binary.
ENTRYPOINT ["/app/gocron"]