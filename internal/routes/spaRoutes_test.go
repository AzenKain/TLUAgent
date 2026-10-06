package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

func spaTestFS() fstest.MapFS {
	return fstest.MapFS{
		"dist/index.html":        &fstest.MapFile{Data: []byte("<html>index</html>")},
		"dist/assets/app.abc.js": &fstest.MapFile{Data: []byte("console.log(1)")},
		"dist/assets/style.css":  &fstest.MapFile{Data: []byte("body{}")},
		"dist/favicon.ico":       &fstest.MapFile{Data: []byte("icon")},
	}
}

func TestSPARoutes_CacheHeaders(t *testing.T) {
	mux := http.NewServeMux()
	RegisterSPARoutes(mux, spaTestFS())

	tests := []struct {
		name      string
		path      string
		wantCache string
	}{
		{name: "root serves index without caching", path: "/", wantCache: "no-cache"},
		{name: "hashed asset is immutable", path: "/assets/app.abc.js", wantCache: "public, max-age=31536000, immutable"},
		{name: "nested asset is immutable", path: "/assets/style.css", wantCache: "public, max-age=31536000, immutable"},
		{name: "spa fallback is no-cache", path: "/some/client/route", wantCache: "no-cache"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("Expected 200 for %s, got %d", tt.path, rec.Code)
			}
			if got := rec.Header().Get("Cache-Control"); got != tt.wantCache {
				t.Fatalf("Cache-Control for %s = %q, want %q", tt.path, got, tt.wantCache)
			}
		})
	}
}

func TestSPARoutes_APIPathsNotFound(t *testing.T) {
	mux := http.NewServeMux()
	RegisterSPARoutes(mux, spaTestFS())

	req := httptest.NewRequest(http.MethodGet, "/api/nonexistent", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("API paths must return 404 from the SPA handler, got %d", rec.Code)
	}
}
