package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static
var staticFiles embed.FS

func StaticFileServer() http.Handler {
	fsys, _ := fs.Sub(staticFiles, "static")
	return http.FileServer(http.FS(fsys))
}
