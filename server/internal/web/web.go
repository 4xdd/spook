// Package web serves the built single-page app, either from the binary or from
// disk when iterating on the frontend.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

//go:embed all:dist
var embedded embed.FS

// Handler serves the SPA. A non-empty devDir reads from disk instead of the
// embedded build so the UI can be rebuilt without recompiling the server.
func Handler(devDir string) (http.Handler, error) {
	var root fs.FS
	if devDir != "" {
		root = os.DirFS(devDir)
	} else {
		sub, err := fs.Sub(embedded, "dist")
		if err != nil {
			return nil, err
		}
		root = sub
	}

	files := http.FileServer(http.FS(root))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" {
			name = "index.html"
		}

		if _, err := fs.Stat(root, name); err != nil {
			// Unknown paths belong to the client router.
			serveIndex(w, r, root)
			return
		}

		if strings.HasPrefix(name, "assets/") {
			// Vite fingerprints asset filenames.
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		files.ServeHTTP(w, r)
	}), nil
}

func serveIndex(w http.ResponseWriter, r *http.Request, root fs.FS) {
	data, err := fs.ReadFile(root, "index.html")
	if err != nil {
		http.Error(w, "web UI not built: run `make build-ui`", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(data)
}
