package static

import (
	"embed"
	"io/fs"
)

//go:embed dist
var embeddedFiles embed.FS

// FS returns a filesystem rooted at the embedded dist/ directory.
// Use this with http.FileServer to serve the dashboard.
func FS() fs.FS {
	sub, err := fs.Sub(embeddedFiles, "dist")
	if err != nil {
		panic("static: failed to sub dist: " + err.Error())
	}
	return sub
}
