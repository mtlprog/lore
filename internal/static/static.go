// Package static provides embedded static files for the web server.
package static

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed favicon.svg og-image.svg
var files embed.FS

// Handler returns an http.Handler that serves static files.
func Handler() http.Handler {
	return http.FileServer(http.FS(files))
}

// FS returns the embedded filesystem.
func FS() fs.FS {
	return files
}
