package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// timeNow is a seam for tests.
var timeNow = func() time.Time { return time.Now().UTC() }

// spaHandler serves the built frontend from staticDir, falling back to
// index.html for client-side routes. With no staticDir (dev mode, where
// Vite serves the frontend), non-API requests get a plain 404.
func (s *Server) spaHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.staticDir == "" {
			http.NotFound(w, r)
			return
		}
		path := filepath.Join(s.staticDir, filepath.Clean("/"+r.URL.Path))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			// Vite emits content-hashed filenames under /assets; cache those hard.
			if strings.HasPrefix(r.URL.Path, "/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			http.ServeFile(w, r, path)
			return
		}
		http.ServeFile(w, r, filepath.Join(s.staticDir, "index.html"))
	})
}
