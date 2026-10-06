package routes

import (
	"io/fs"
	"net/http"
	"strings"
)

const (
	indexCacheControl       = "no-cache"
	assetsCacheControl      = "public, max-age=31536000, immutable"
	assetsPathPrefix        = "assets/"
	indexHTMLPath           = "index.html"
	fallbackHTMLContentType = "text/html; charset=utf-8"
)

func RegisterSPARoutes(mux *http.ServeMux, embeddedDist fs.FS) {
	distFS, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		distFS = embeddedDist
	}

	fileServer := http.FileServer(http.FS(distFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		cleanPath := strings.TrimPrefix(r.URL.Path, "/")
		if cleanPath == "" {
			cleanPath = indexHTMLPath
		}

		f, err := distFS.Open(cleanPath)
		if err == nil {
			stat, statErr := f.Stat()
			f.Close()
			if statErr == nil && !stat.IsDir() {
				if strings.HasPrefix(cleanPath, assetsPathPrefix) {
					w.Header().Set("Cache-Control", assetsCacheControl)
				} else if cleanPath == indexHTMLPath {
					w.Header().Set("Cache-Control", indexCacheControl)
				}
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		indexBytes, err := fs.ReadFile(distFS, indexHTMLPath)
		if err != nil {
			http.Error(w, "SPA index.html not found", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", fallbackHTMLContentType)
		w.Header().Set("Cache-Control", indexCacheControl)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(indexBytes)
	})
}
