//go:build dev

package main

import (
	"net/http"
)

func staticFS() http.FileSystem {
	return nil
}


