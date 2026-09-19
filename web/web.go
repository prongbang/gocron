// Package web embeds the dashboard built by `bun run build` into the gocron binary.
package web

import (
	"embed"
	"io/fs"
)

// build/.gitkeep is committed so this compiles even when the UI was not built.
//
//go:embed all:build
var build embed.FS

// FS returns the built dashboard, or nil when `bun run build` did not run before `go build`.
func FS() fs.FS {
	ui, _ := fs.Sub(build, "build")
	if _, err := fs.Stat(ui, "index.html"); err != nil {
		return nil
	}
	return ui
}
