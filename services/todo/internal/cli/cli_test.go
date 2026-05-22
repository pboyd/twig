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

	code := runTask([]string{})
	if code != 0 {
		t.Fatalf("runTask (default list): exit code %d", code)
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

// --- T006: runTask dispatch tests (US2) ---

func TestRunTaskFlagArgsInvokesList(t *testing.T) {
	svc := newFakeTaskService()
	mux := http.NewServeMux()
	path, handler := taskv1connect.NewTaskServiceHandler(svc)
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("TODO_ADDR", srv.URL)
	t.Setenv("TODO_API_KEY", "testkey")

	r, w, _ := os.Pipe()
	oldOut := os.Stdout
	os.Stdout = w
	code := runTask([]string{"--all"})
	w.Close()
	os.Stdout = oldOut
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	out := string(buf[:n])

	if code != 0 {
		t.Fatalf("expected exit 0 for --all flag, got %d; output: %s", code, out)
	}
}

func TestRunTaskEmptyArgsInvokesList(t *testing.T) {
	svc := newFakeTaskService()
	mux := http.NewServeMux()
	path, handler := taskv1connect.NewTaskServiceHandler(svc)
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("TODO_ADDR", srv.URL)
	t.Setenv("TODO_API_KEY", "testkey")

	r, w, _ := os.Pipe()
	oldOut := os.Stdout
	os.Stdout = w
	code := runTask([]string{})
	w.Close()
	os.Stdout = oldOut
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	out := string(buf[:n])

	if code != 0 {
		t.Fatalf("expected exit 0 for empty args, got %d; output: %s", code, out)
	}
	// Empty service returns "no tasks"
	if !strings.Contains(out, "no tasks") {
		t.Errorf("expected list output, got: %q", out)
	}
}

func TestRunTaskListSubcommandUnknown(t *testing.T) {
	t.Setenv("TODO_API_KEY", "testkey")
	t.Setenv("TODO_ADDR", "http://localhost:19999")

	r, w, _ := os.Pipe()
	oldErr := os.Stderr
	os.Stderr = w
	code := runTask([]string{"list"})
	w.Close()
	os.Stderr = oldErr
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	errOut := string(buf[:n])

	if code == 0 {
		t.Fatal("expected non-zero exit code for 'list' subcommand")
	}
	if !strings.Contains(errOut, "unknown subcommand") {
		t.Errorf("expected 'unknown subcommand' in stderr, got: %q", errOut)
	}
}

// --- T011: help routing tests ---

func captureStdout(fn func()) string {
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	return string(buf[:n])
}

func TestRunHelpNoArgs(t *testing.T) {
	out := captureStdout(func() {
		code := Run([]string{"help"})
		if code != 0 {
			t.Errorf("Run(help): expected exit 0, got %d", code)
		}
	})
	if !strings.Contains(out, "task") || !strings.Contains(out, "pom") || !strings.Contains(out, "plan") {
		t.Errorf("Run(help): expected command listing in output, got: %q", out)
	}
}

func TestRunHelpFlag(t *testing.T) {
	out := captureStdout(func() {
		code := Run([]string{"--help"})
		if code != 0 {
			t.Errorf("Run(--help): expected exit 0, got %d", code)
		}
	})
	if !strings.Contains(out, "task") || !strings.Contains(out, "pom") || !strings.Contains(out, "plan") {
		t.Errorf("Run(--help): expected command listing in output, got: %q", out)
	}
}

func TestRunHelpTask(t *testing.T) {
	out := captureStdout(func() {
		code := Run([]string{"help", "task"})
		if code != 0 {
			t.Errorf("Run(help task): expected exit 0, got %d", code)
		}
	})
	if !strings.Contains(out, "add") || !strings.Contains(out, "complete") {
		t.Errorf("Run(help task): expected task subcommands in output, got: %q", out)
	}
}

func TestRunHelpUnknown(t *testing.T) {
	r, w, _ := os.Pipe()
	oldErr := os.Stderr
	os.Stderr = w
	code := Run([]string{"help", "unknown"})
	w.Close()
	os.Stderr = oldErr
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	errOut := string(buf[:n])

	if code == 0 {
		t.Error("Run(help unknown): expected non-zero exit code")
	}
	if !strings.Contains(errOut, "unknown command") {
		t.Errorf("Run(help unknown): expected 'unknown command' in stderr, got: %q", errOut)
	}
}
