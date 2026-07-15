//go:build !dev

package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:web/build
var webFS embed.FS

func staticFS() http.FileSystem {
	sub, err := fs.Sub(webFS, "web/build")
	if err != nil {
		panic("failed to get embedded FS: " + err.Error())
	}
	return http.FS(sub)
}
