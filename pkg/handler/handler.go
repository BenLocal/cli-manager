package handler

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// NewAppHandler serves the built frontend bundle and supports SPA history routing.
func NewAppHandler() (http.Handler, error) {
	staticFS, err := embeddedUIFS()
	if err != nil {
		return nil, err
	}

	fileServer := http.FileServer(http.FS(staticFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleanPath := path.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))
		relativePath := strings.TrimPrefix(cleanPath, "/")
		if relativePath == "" || relativePath == "." {
			relativePath = "index.html"
		}

		file, err := staticFS.Open(relativePath)
		if err == nil {
			_ = file.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		if strings.Contains(path.Base(relativePath), ".") {
			http.NotFound(w, r)
			return
		}

		index, err := fs.ReadFile(staticFS, "index.html")
		if err != nil {
			http.Error(w, "embedded index.html not found", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	}), nil
}
