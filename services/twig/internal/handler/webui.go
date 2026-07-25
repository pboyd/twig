package handler

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
)

// WebUI serves the built twig-web single-page app, with client-side-route
// fallback to index.html.
type WebUI struct {
	available bool
	dir       string
	fileSrv   http.Handler
}

// NewWebUI returns a handler serving the SPA build in dir. If dir has no
// index.html, the returned handler responds 404 to every request instead of
// failing to start — this keeps the server usable without a frontend build.
func NewWebUI(dir string) *WebUI {
	w := &WebUI{dir: dir}
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err == nil {
		w.available = true
		w.fileSrv = http.FileServer(http.Dir(dir))
	}
	return w
}

func (w *WebUI) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if !w.available {
		http.Error(rw, "web UI unavailable", http.StatusNotFound)
		return
	}

	if _, err := os.Stat(filepath.Join(w.dir, filepath.FromSlash(r.URL.Path))); err != nil {
		// No file on disk for this path. If it looks like a client-side
		// route (no file extension), fall back to index.html so the SPA
		// router can take over. Otherwise it's a genuinely missing asset.
		if path.Ext(r.URL.Path) == "" {
			http.ServeFile(rw, r, filepath.Join(w.dir, "index.html"))
			return
		}
		http.NotFound(rw, r)
		return
	}

	w.fileSrv.ServeHTTP(rw, r)
}
