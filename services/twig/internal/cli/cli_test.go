package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	taskv1connect "github.com/pboyd/twig/services/twig/gen/task/v1/taskv1connect"
)

// --- T018: unknown profile error (US4) ---

func TestLoadConfigUnknownProfile(t *testing.T) {
	xdgHome := writeConfigFile(t, `
api_url = "http://root.example.com"
api_key = "root-token"

[profile.home]
api_url = "http://home.example.com"
api_key = "home-token"
`)
	t.Setenv("XDG_CONFIG_HOME", xdgHome)
	t.Setenv("TWIG_ADDR", "")
	t.Setenv("TWIG_API_KEY", "")

	// Capture stderr to ensure no client is constructed / no request is made.
	r, w, _ := os.Pipe()
	oldErr := os.Stderr
	os.Stderr = w

	code := runTask("bogus", []string{})

	w.Close()
	os.Stderr = oldErr
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	errOut := string(buf[:n])

	if code == 0 {
		t.Error("expected non-zero exit code for unknown profile")
	}
	if !strings.Contains(errOut, "bogus") {
		t.Errorf("error should mention the profile name 'bogus', got: %q", errOut)
	}
	if !strings.Contains(errOut, "config.toml") {
		t.Errorf("error should mention the config path, got: %q", errOut)
	}
}

// --- T016: profile-name resolution precedence tests ---

func TestResolveProfileName(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		flagSet   bool
		envValue  string
		want      string
	}{
		{
			name:      "flag value wins when set",
			flagValue: "work",
			flagSet:   true,
			envValue:  "home",
			want:      "work",
		},
		{
			name:      "TWIG_PROFILE used when flag absent",
			flagValue: "",
			flagSet:   false,
			envValue:  "home",
			want:      "home",
		},
		{
			name:      "empty TWIG_PROFILE treated as unset (default)",
			flagValue: "",
			flagSet:   false,
			envValue:  "",
			want:      "",
		},
		{
			name:      "flag default overrides TWIG_PROFILE=home",
			flagValue: "default",
			flagSet:   true,
			envValue:  "home",
			want:      "default",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TWIG_PROFILE", tc.envValue)
			got := ResolveProfileName(tc.flagValue, tc.flagSet)
			if got != tc.want {
				t.Errorf("ResolveProfileName(%q, %v) = %q, want %q", tc.flagValue, tc.flagSet, got, tc.want)
			}
		})
	}
}

// --- T005: global --profile flag extractor tests ---

func TestExtractProfileFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantName string
		wantRest []string
		wantErr  bool
	}{
		{
			name:     "no flag returns empty name and unchanged args",
			args:     []string{"task"},
			wantName: "",
			wantRest: []string{"task"},
		},
		{
			name:     "two-arg form --profile home",
			args:     []string{"--profile", "home", "task"},
			wantName: "home",
			wantRest: []string{"task"},
		},
		{
			name:     "equals form --profile=home",
			args:     []string{"--profile=home", "task"},
			wantName: "home",
			wantRest: []string{"task"},
		},
		{
			name:     "no subcommand after --profile (TUI launch)",
			args:     []string{"--profile", "home"},
			wantName: "home",
			wantRest: []string{},
		},
		{
			name:    "--profile with no value is a usage error",
			args:    []string{"--profile"},
			wantErr: true,
		},
		{
			name:    "--profile= with empty value is a usage error",
			args:    []string{"--profile="},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			name, _, rest, err := ExtractProfileFlag(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if name != tc.wantName {
				t.Errorf("name = %q, want %q", name, tc.wantName)
			}
			if !reflect.DeepEqual(rest, tc.wantRest) {
				t.Errorf("rest = %v, want %v", rest, tc.wantRest)
			}
		})
	}
}

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

	t.Setenv("TWIG_ADDR", srv.URL)
	t.Setenv("TWIG_API_KEY", wantKey)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	code := runTask("", []string{})
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
	t.Setenv("TWIG_API_KEY", "")
	t.Setenv("TWIG_ADDR", "http://localhost:19999") // won't be reached
	// Point XDG_CONFIG_HOME at an empty temp dir so there's no config file.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// Capture stderr.
	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	code := runTask("", []string{"list"})

	w.Close()
	os.Stderr = origStderr
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	if code == 0 {
		t.Error("expected non-zero exit code when TWIG_API_KEY is unset")
	}
	if !strings.Contains(output, "TWIG_API_KEY") {
		t.Errorf("stderr should mention TWIG_API_KEY, got: %q", output)
	}
	if !strings.Contains(output, "api_key") {
		t.Errorf("stderr should mention config file api_key, got: %q", output)
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

	t.Setenv("TWIG_ADDR", srv.URL)
	t.Setenv("TWIG_API_KEY", "testkey")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	r, w, _ := os.Pipe()
	oldOut := os.Stdout
	os.Stdout = w
	code := runTask("", []string{"--all"})
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

	t.Setenv("TWIG_ADDR", srv.URL)
	t.Setenv("TWIG_API_KEY", "testkey")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	r, w, _ := os.Pipe()
	oldOut := os.Stdout
	os.Stdout = w
	code := runTask("", []string{})
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
	t.Setenv("TWIG_API_KEY", "testkey")
	t.Setenv("TWIG_ADDR", "http://localhost:19999")
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	r, w, _ := os.Pipe()
	oldErr := os.Stderr
	os.Stderr = w
	code := runTask("", []string{"list"})
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
		code := Run("", []string{"help"})
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
		code := Run("", []string{"--help"})
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
		code := Run("", []string{"help", "task"})
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
	code := Run("", []string{"help", "unknown"})
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

// writeConfigFile writes a TOML config to a temp dir and returns its XDG_CONFIG_HOME.
func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	cfgDir := filepath.Join(dir, "twig")
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return dir // this is what XDG_CONFIG_HOME should be set to
}

// TestConfigFileAPIKeyUsedWhenEnvUnset verifies config file api_key is used when env is unset.
func TestConfigFileAPIKeyUsedWhenEnvUnset(t *testing.T) {
	svc := newFakeTaskService()
	mux := http.NewServeMux()
	path, handler := taskv1connect.NewTaskServiceHandler(svc)

	var gotAuth string
	mux.Handle(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		handler.ServeHTTP(w, r)
	}))

	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("TWIG_API_KEY", "")
	t.Setenv("TWIG_ADDR", "")
	xdgHome := writeConfigFile(t, fmt.Sprintf(`api_url = %q
api_key = "config-key-xyz"
`, srv.URL))
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	code := runTask("", []string{})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if gotAuth != "Bearer config-key-xyz" {
		t.Errorf("expected config-file key, got %q", gotAuth)
	}
}

// TestEnvKeyWinsOverConfigKey verifies env var takes precedence over config file.
func TestEnvKeyWinsOverConfigKey(t *testing.T) {
	svc := newFakeTaskService()
	mux := http.NewServeMux()
	path, handler := taskv1connect.NewTaskServiceHandler(svc)

	var gotAuth string
	mux.Handle(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		handler.ServeHTTP(w, r)
	}))

	srv := httptest.NewServer(mux)
	defer srv.Close()

	t.Setenv("TWIG_API_KEY", "env-key-wins")
	t.Setenv("TWIG_ADDR", srv.URL)
	xdgHome := writeConfigFile(t, `api_key = "config-key-should-lose"`)
	t.Setenv("XDG_CONFIG_HOME", xdgHome)

	code := runTask("", []string{})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	if gotAuth != "Bearer env-key-wins" {
		t.Errorf("expected env key to win, got %q", gotAuth)
	}
}
