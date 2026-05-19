package cli

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	taskv1connect "github.com/pboyd/todo/services/todo/gen/task/v1/taskv1connect"
)

// TestAPIKeyAttachedToRequests verifies that task commands send the
// Authorization: Bearer header when TODO_API_KEY is set.
func TestAPIKeyAttachedToRequests(t *testing.T) {
	const wantKey = "testkey1234"

	var gotAuth string
	mux := http.NewServeMux()
	fakeSvc := &fakeTaskService{}
	path, h := taskv1connect.NewTaskServiceHandler(fakeSvc)
	mux.Handle(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		h.ServeHTTP(w, r)
	}))

	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("TODO_ADDR", srv.URL)
	t.Setenv("TODO_API_KEY", wantKey)

	code := runTask([]string{"list"})
	if code != 0 {
		t.Fatalf("runTask list: exit code %d", code)
	}

	want := "Bearer " + wantKey
	if gotAuth != want {
		t.Errorf("Authorization = %q, want %q", gotAuth, want)
	}
}

// TestMissingAPIKey verifies that a task command fails with a non-zero exit
// code and a helpful message when TODO_API_KEY is not set.
func TestMissingAPIKey(t *testing.T) {
	t.Setenv("TODO_API_KEY", "")
	t.Setenv("TODO_ADDR", "http://localhost:19999") // won't be reached

	// Capture stderr.
	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	code := runTask([]string{"list"})

	w.Close()
	os.Stderr = origStderr
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if code == 0 {
		t.Error("expected non-zero exit code when TODO_API_KEY is unset")
	}
	if !strings.Contains(output, "TODO_API_KEY") {
		t.Errorf("stderr should mention TODO_API_KEY, got: %q", output)
	}
}
