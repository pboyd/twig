package handler_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/pboyd/twig/services/twig/internal/handler"
)

func makeCLIFixture(t *testing.T, binaryContent []byte, version string) string {
	t.Helper()
	dir := t.TempDir()
	if binaryContent != nil {
		if err := os.WriteFile(filepath.Join(dir, "twig"), binaryContent, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if version != "" {
		if err := os.WriteFile(filepath.Join(dir, "version"), []byte(version), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestServeDownload_OK(t *testing.T) {
	content := []byte("fake-twig-binary-content")
	dir := makeCLIFixture(t, content, "v1.2.3")
	c := handler.NewCLIBinary(dir)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/cli/download", nil)
	c.ServeDownload(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/octet-stream" {
		t.Errorf("Content-Type = %q, want application/octet-stream", ct)
	}
	if cd := rr.Header().Get("Content-Disposition"); cd != `attachment; filename="twig"` {
		t.Errorf("Content-Disposition = %q", cd)
	}
	if body := rr.Body.Bytes(); string(body) != string(content) {
		t.Errorf("body mismatch: got %q, want %q", body, content)
	}
}

func TestServeDownload_Unavailable(t *testing.T) {
	dir := makeCLIFixture(t, nil, "")
	c := handler.NewCLIBinary(dir)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/cli/download", nil)
	c.ServeDownload(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
}

func TestServeInfo_OK(t *testing.T) {
	content := []byte("fake-twig-binary")
	dir := makeCLIFixture(t, content, "v2.0.0\n")
	c := handler.NewCLIBinary(dir)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/cli/info", nil)
	c.ServeInfo(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var info map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&info); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if info["version"] != "v2.0.0" {
		t.Errorf("version = %v, want v2.0.0", info["version"])
	}
	if info["filename"] != "twig" {
		t.Errorf("filename = %v, want twig", info["filename"])
	}
	if info["os"] != "linux" {
		t.Errorf("os = %v, want linux", info["os"])
	}
	if info["arch"] != "amd64" {
		t.Errorf("arch = %v, want amd64", info["arch"])
	}
	if info["label"] != "Linux (x86-64)" {
		t.Errorf("label = %v, want Linux (x86-64)", info["label"])
	}

	sum := sha256.Sum256(content)
	wantHash := hex.EncodeToString(sum[:])
	if info["sha256"] != wantHash {
		t.Errorf("sha256 = %v, want %v", info["sha256"], wantHash)
	}

	if size, ok := info["size"].(float64); !ok || int64(size) != int64(len(content)) {
		t.Errorf("size = %v, want %d", info["size"], len(content))
	}
}

func TestServeInfo_Unavailable(t *testing.T) {
	dir := makeCLIFixture(t, nil, "")
	c := handler.NewCLIBinary(dir)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/cli/info", nil)
	c.ServeInfo(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["error"] != "binary unavailable" {
		t.Errorf("error = %q, want binary unavailable", resp["error"])
	}
}
