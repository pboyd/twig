package config_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pboyd/twig/services/twig/internal/config"
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
			name: "profile block decodes into Profiles map",
			setup: func(t *testing.T) string {
				return writeConfig(t, `
api_url = "http://root.example.com"
api_key = "root-token"

[profile.home]
api_url = "http://home.example.com"
api_key = "home-token"
`)
			},
			wantCfg: config.Config{
				APIURL: "http://root.example.com",
				APIKey: "root-token",
				Profiles: map[string]config.Profile{
					"home": {APIURL: "http://home.example.com", APIKey: "home-token"},
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
			if !reflect.DeepEqual(got, tc.wantCfg) {
				t.Errorf("got %+v, want %+v", got, tc.wantCfg)
			}
		})
	}
}

func TestProfile(t *testing.T) {
	rootCfg := config.Config{
		APIURL: "http://root.example.com",
		APIKey: "root-token",
		Pomodoro: config.PomodoroConfig{
			OnComplete: "notify.sh",
		},
		Profiles: map[string]config.Profile{
			"home": {APIURL: "http://home.example.com", APIKey: "home-token"},
			"slim": {APIKey: "slim-token"}, // no api_url
		},
	}

	tests := []struct {
		name    string
		input   string
		wantCfg config.Config
		wantOk  bool
	}{
		{
			name:    "empty name returns root with ok=true",
			input:   "",
			wantCfg: rootCfg,
			wantOk:  true,
		},
		{
			name:    "default name returns root with ok=true",
			input:   "default",
			wantCfg: rootCfg,
			wantOk:  true,
		},
		{
			name: "named profile returns its credentials with ok=true",
			input: "home",
			wantCfg: config.Config{
				APIURL:   "http://home.example.com",
				APIKey:   "home-token",
				Pomodoro: rootCfg.Pomodoro,
				Profiles: rootCfg.Profiles,
			},
			wantOk: true,
		},
		{
			name: "profile with no api_url returns empty APIURL (not root URL)",
			input: "slim",
			wantCfg: config.Config{
				APIURL:   "",
				APIKey:   "slim-token",
				Pomodoro: rootCfg.Pomodoro,
				Profiles: rootCfg.Profiles,
			},
			wantOk: true,
		},
		{
			name:   "unknown profile returns false",
			input:  "bogus",
			wantOk: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := rootCfg.Profile(tc.input)
			if ok != tc.wantOk {
				t.Fatalf("Profile(%q) ok = %v, want %v", tc.input, ok, tc.wantOk)
			}
			if ok && !reflect.DeepEqual(got, tc.wantCfg) {
				t.Errorf("Profile(%q) = %+v, want %+v", tc.input, got, tc.wantCfg)
			}
		})
	}
}

func TestProfileSelectResolveOrder(t *testing.T) {
	// Profile("home").Resolve() with env vars set: env wins over profile credentials.
	cfg := config.Config{
		APIURL: "http://root.example.com",
		APIKey: "root-token",
		Profiles: map[string]config.Profile{
			"home": {APIURL: "http://home.example.com", APIKey: "home-token"},
			"slim": {APIKey: "slim-token"}, // no api_url → should fall back to built-in default
		},
	}

	t.Run("env overrides selected profile credentials", func(t *testing.T) {
		t.Setenv("TWIG_ADDR", "http://env.example.com")
		t.Setenv("TWIG_API_KEY", "env-key")
		selected, ok := cfg.Profile("home")
		if !ok {
			t.Fatal("Profile(home) returned ok=false")
		}
		got := selected.Resolve()
		if got.APIURL != "http://env.example.com" {
			t.Errorf("APIURL = %q, want env value", got.APIURL)
		}
		if got.APIKey != "env-key" {
			t.Errorf("APIKey = %q, want env value", got.APIKey)
		}
	})

	t.Run("profile with no api_url and no env gets built-in default", func(t *testing.T) {
		t.Setenv("TWIG_ADDR", "")
		t.Setenv("TWIG_API_KEY", "")
		selected, ok := cfg.Profile("slim")
		if !ok {
			t.Fatal("Profile(slim) returned ok=false")
		}
		got := selected.Resolve()
		if got.APIURL != "http://localhost:8080" {
			t.Errorf("APIURL = %q, want built-in default", got.APIURL)
		}
	})
}

func TestProfileBackwardCompat(t *testing.T) {
	// A config with no profile blocks: Profiles is nil; Profile("") and
	// Profile("default") return root unchanged.
	noProfilesCfg := config.Config{
		APIURL: "http://root.example.com",
		APIKey: "root-token",
	}

	tests := []struct {
		name    string
		cfg     config.Config
		input   string
		wantCfg config.Config
		wantOk  bool
	}{
		{
			name:    "no profiles, empty name returns root with ok=true",
			cfg:     noProfilesCfg,
			input:   "",
			wantCfg: noProfilesCfg,
			wantOk:  true,
		},
		{
			name:    "no profiles, default returns root with ok=true",
			cfg:     noProfilesCfg,
			input:   "default",
			wantCfg: noProfilesCfg,
			wantOk:  true,
		},
		{
			// Even if a [profile.default] table exists, Profile("default")
			// must return root credentials (the root is authoritative for "default").
			name: "profile.default table does not override default selection",
			cfg: config.Config{
				APIURL: "http://root.example.com",
				APIKey: "root-token",
				Profiles: map[string]config.Profile{
					"default": {APIURL: "http://other.example.com", APIKey: "other-token"},
				},
			},
			input: "default",
			wantCfg: config.Config{
				APIURL: "http://root.example.com",
				APIKey: "root-token",
				Profiles: map[string]config.Profile{
					"default": {APIURL: "http://other.example.com", APIKey: "other-token"},
				},
			},
			wantOk: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := tc.cfg.Profile(tc.input)
			if ok != tc.wantOk {
				t.Fatalf("Profile(%q) ok = %v, want %v", tc.input, ok, tc.wantOk)
			}
			if ok && !reflect.DeepEqual(got, tc.wantCfg) {
				t.Errorf("Profile(%q) = %+v, want %+v", tc.input, got, tc.wantCfg)
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
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Resolve() = %+v, want %+v", got, tc.want)
			}
		})
	}
}
