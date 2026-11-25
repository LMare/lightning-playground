package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed static
var staticFS embed.FS

// StaticHandler serves the embedded assets under /asset/ (no directory listing).
func StaticHandler() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	files := http.FileServerFS(sub)
	return http.StripPrefix("/asset/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p == "" || strings.HasSuffix(p, "/") {
			http.NotFound(w, r)
			return
		}
		if info, err := fs.Stat(sub, strings.TrimPrefix(p, "/")); err == nil && info.IsDir() {
			http.NotFound(w, r)
			return
		}
		files.ServeHTTP(w, r)
	}))
}
