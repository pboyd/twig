package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pboyd/todo/services/todo/internal/config"
)

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string // returns path
		wantErr string
		wantCfg config.Config
	}{
		{
			name: "missing file returns zero config",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.toml")
			},
		},
		{
			name: "empty file returns zero config",
			setup: func(t *testing.T) string {
				return writeConfig(t, "")
			},
		},
		{
			name: "all keys set",
			setup: func(t *testing.T) string {
				return writeConfig(t, `
api_url = "http://example.com:8080"
api_key = "mytoken"

[pomodoro]
on_start    = "start.sh"
on_cancel   = "cancel.sh"
on_complete = "complete.sh"
`)
			},
			wantCfg: config.Config{
				APIURL: "http://example.com:8080",
				APIKey: "mytoken",
				Pomodoro: config.PomodoroConfig{
					OnStart:    "start.sh",
					OnCancel:   "cancel.sh",
					OnComplete: "complete.sh",
				},
			},
		},
		{
			name: "malformed TOML returns error mentioning path",
			setup: func(t *testing.T) string {
				return writeConfig(t, "this is = not valid toml [[[")
			},
			wantErr: "config: failed to parse",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.setup(t)
			got, err := config.Load(path)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.wantErr)
				}
				if !strings.Contains(err.Error(), path) {
					t.Errorf("error %q should mention path %q", err.Error(), path)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantCfg {
				t.Errorf("got %+v, want %+v", got, tc.wantCfg)
			}
		})
	}
}

func TestResolve(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Config
		envAddr string
		envKey  string
		want    config.Config
	}{
		{
			name:    "env vars override config values",
			cfg:     config.Config{APIURL: "http://config.host", APIKey: "config-key"},
			envAddr: "http://env.host",
			envKey:  "env-key",
			want:    config.Config{APIURL: "http://env.host", APIKey: "env-key"},
		},
		{
			name: "config used when env unset",
			cfg:  config.Config{APIURL: "http://config.host", APIKey: "config-key"},
			want: config.Config{APIURL: "http://config.host", APIKey: "config-key"},
		},
		{
			name: "default URL when both empty",
			cfg:  config.Config{},
			want: config.Config{APIURL: "http://localhost:8080"},
		},
		{
			name: "empty APIKey left empty when both sources unset",
			cfg:  config.Config{APIURL: "http://localhost:8080"},
			want: config.Config{APIURL: "http://localhost:8080", APIKey: ""},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envAddr != "" {
				t.Setenv("TWIG_ADDR", tc.envAddr)
			} else {
				t.Setenv("TWIG_ADDR", "")
			}
			if tc.envKey != "" {
				t.Setenv("TWIG_API_KEY", tc.envKey)
			} else {
				t.Setenv("TWIG_API_KEY", "")
			}
			got := tc.cfg.Resolve()
			if got != tc.want {
				t.Errorf("Resolve() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
