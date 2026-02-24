package frontend

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist/*
var embeddedFsys embed.FS

func BrowserFS() fs.FS {
	if fs, err := fs.Sub(embeddedFsys, "dist/frontend/browser"); err != nil {
		panic(err)
	} else {
		return fs
	}
}

func NewHandler() http.Handler {
	fsys := BrowserFS()
	server := http.FileServer(http.FS(fsys))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check if the requested file exists and use index.html if it does not.
		if _, err := fs.Stat(fsys, r.URL.Path[1:]); err != nil {
			http.StripPrefix(r.URL.Path, server).ServeHTTP(w, r)
		} else {
			server.ServeHTTP(w, r)
		}
	})
}
