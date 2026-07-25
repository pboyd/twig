package handler

import (
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// WebUI serves the built twig-web single-page app, with client-side-route
// fallback to index.html.
type WebUI struct {
	available bool
	fsys      http.FileSystem
}

// NewWebUI returns a handler serving the SPA build in dir. If dir has no
// index.html, the returned handler responds 404 to every request instead of
// failing to start — this keeps the server usable without a frontend build.
func NewWebUI(dir string) *WebUI {
	w := &WebUI{}
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err == nil {
		w.available = true
		w.fsys = http.Dir(dir)
	}
	return w
}

func (w *WebUI) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	if !w.available {
		http.Error(rw, "web UI unavailable", http.StatusNotFound)
		return
	}

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		rw.Header().Set("Allow", "GET, HEAD")
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := path.Clean("/" + r.URL.Path)
	f, err := w.fsys.Open(name)
	if err != nil {
		if os.IsNotExist(err) && strings.HasPrefix(name, "/assets/") {
			http.NotFound(rw, r)
			return
		}
		serveIndex(rw, r, w.fsys)
		return
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		http.Error(rw, "internal server error", http.StatusInternalServerError)
		return
	}

	if st.IsDir() {
		serveIndex(rw, r, w.fsys)
		return
	}

	if strings.HasPrefix(name, "/assets/") {
		rw.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		rw.Header().Set("Cache-Control", "no-cache")
	}

	http.ServeContent(rw, r, st.Name(), st.ModTime(), f.(io.ReadSeeker))
}

func serveIndex(rw http.ResponseWriter, r *http.Request, fsys http.FileSystem) {
	f, err := fsys.Open("/index.html")
	if err != nil {
		http.Error(rw, "internal server error", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	st, err := f.Stat()
	if err != nil {
		http.Error(rw, "internal server error", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Cache-Control", "no-cache")
	http.ServeContent(rw, r, st.Name(), st.ModTime(), f.(io.ReadSeeker))
}
