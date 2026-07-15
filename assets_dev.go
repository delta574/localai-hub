//go:build dev

package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

func staticFS() http.FileSystem {
	return nil
}

func staticHandler() http.Handler {
	remote, _ := url.Parse("http://localhost:5173")
	proxy := httputil.NewSingleHostReverseProxy(remote)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		proxy.ServeHTTP(w, r)
	})
}
