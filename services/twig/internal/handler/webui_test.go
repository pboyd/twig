package handler_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pboyd/twig/services/twig/internal/handler"
)

func makeWebUIFixture(t *testing.T, withIndex bool) string {
	t.Helper()
	dir := t.TempDir()
	if withIndex {
		if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>index</html>"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log('app')"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestWebUI_ServesRealFile(t *testing.T) {
	dir := makeWebUIFixture(t, true)
	w := handler.NewWebUI(dir)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	w.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if body := rr.Body.String(); body != "console.log('app')" {
		t.Errorf("body = %q, want app.js content", body)
	}
}

func TestWebUI_FallsBackToIndexForClientRoute(t *testing.T) {
	dir := makeWebUIFixture(t, true)
	w := handler.NewWebUI(dir)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tasks/abc123", nil)
	w.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if body := rr.Body.String(); body != "<html>index</html>" {
		t.Errorf("body = %q, want index.html content", body)
	}
}

func TestWebUI_MissingAssetReturns404(t *testing.T) {
	dir := makeWebUIFixture(t, true)
	w := handler.NewWebUI(dir)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/missing.js", nil)
	w.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
}

func TestWebUI_UnavailableReturns404(t *testing.T) {
	dir := makeWebUIFixture(t, false)
	w := handler.NewWebUI(dir)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rr.Code)
	}
	if body := rr.Body.String(); body != "web UI unavailable\n" {
		t.Errorf("body = %q, want %q", body, "web UI unavailable\n")
	}
}

func TestWebUI_ServesIndexForRoot(t *testing.T) {
	dir := makeWebUIFixture(t, true)
	w := handler.NewWebUI(dir)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if body := rr.Body.String(); body != "<html>index</html>" {
		t.Errorf("body = %q, want index.html content", body)
	}
}

func TestWebUI_AssetsDirNotListing(t *testing.T) {
	dir := makeWebUIFixture(t, true)
	w := handler.NewWebUI(dir)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/", nil)
	w.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if strings.Contains(body, "app.js") {
		t.Errorf("directory listing leaked asset names: %s", body)
	}
	if body != "<html>index</html>" {
		t.Errorf("body = %q, want index.html content", body)
	}
}

func TestWebUI_PostReturns405(t *testing.T) {
	dir := makeWebUIFixture(t, true)
	w := handler.NewWebUI(dir)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/tasks", nil)
	w.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
	if allow := rr.Header().Get("Allow"); allow != "GET, HEAD" {
		t.Errorf("Allow header = %q, want %q", allow, "GET, HEAD")
	}
}

func TestWebUI_AssetCacheControlImmutable(t *testing.T) {
	dir := makeWebUIFixture(t, true)
	w := handler.NewWebUI(dir)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	w.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	cc := rr.Header().Get("Cache-Control")
	if !strings.Contains(cc, "immutable") {
		t.Errorf("Cache-Control = %q, want immutable", cc)
	}
}

func TestWebUI_ClientRouteCacheControlNoCache(t *testing.T) {
	dir := makeWebUIFixture(t, true)
	w := handler.NewWebUI(dir)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tasks/abc", nil)
	w.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	cc := rr.Header().Get("Cache-Control")
	if cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", cc)
	}
}

func TestWebUI_ClientRouteWithDot(t *testing.T) {
	dir := makeWebUIFixture(t, true)
	w := handler.NewWebUI(dir)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/tasks/v1.2", nil)
	w.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if body := rr.Body.String(); body != "<html>index</html>" {
		t.Errorf("body = %q, want index.html content", body)
	}
}
