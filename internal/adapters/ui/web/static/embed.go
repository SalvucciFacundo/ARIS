package static

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist/*
var distFS embed.FS

// FS returns an http.FileSystem backed by the embedded dist directory.
func FS() http.FileSystem {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	return http.FS(sub)
}
