package handler_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
