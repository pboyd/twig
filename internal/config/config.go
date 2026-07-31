package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	APIURL   string             `toml:"api_url"`
	APIKey   string             `toml:"api_key"`
	Pomodoro PomodoroConfig     `toml:"pomodoro"`
	Plan     PlanConfig         `toml:"plan"`
	Profiles map[string]Profile `toml:"profile"`
}

type Profile struct {
	APIURL string `toml:"api_url"`
	APIKey string `toml:"api_key"`
}

type PomodoroConfig struct {
	OnStart    string `toml:"on_start"`
	OnCancel   string `toml:"on_cancel"`
	OnComplete string `toml:"on_complete"`
}

type PlanConfig struct {
	OnEventStart string `toml:"on_event_start"`
	OnEventEnd   string `toml:"on_event_end"`
	OnTaskStart  string `toml:"on_task_start"`
	OnTaskEnd    string `toml:"on_task_end"`
}

// Enabled reports whether any plan hook is configured. Gates the TUI ticker
// and background fetch so an unconfigured user pays nothing (SC-006).
func (p PlanConfig) Enabled() bool {
	return p.OnEventStart != "" || p.OnEventEnd != "" || p.OnTaskStart != "" || p.OnTaskEnd != ""
}

// DefaultPath returns the platform-standard path for the config file:
// $XDG_CONFIG_HOME/twig/config.toml (falls back to ~/.config/twig/config.toml).
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config: cannot determine user config dir: %w", err)
	}
	return dir + "/twig/config.toml", nil
}

// Load reads and parses the TOML config file at path.
// A missing file is not an error — it returns a zero Config{}.
func Load(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("config: failed to read %s: %w", path, err)
	}
	defer f.Close()

	var cfg Config
	if _, err := toml.NewDecoder(f).Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("config: failed to parse %s: %w", path, err)
	}
	return cfg, nil
}

// Profile returns the effective Config for the named profile.
// For "" or "default", it returns c unchanged with ok=true (root credentials).
// For a named profile, it returns a copy of c with APIURL and APIKey replaced
// by the profile's values (no fallback to root credentials); Pomodoro is always
// from the root. Returns Config{}, false when the name is not found.
func (c Config) Profile(name string) (Config, bool) {
	if name == "" || name == "default" {
		return c, true
	}
	p, ok := c.Profiles[name]
	if !ok {
		return Config{}, false
	}
	out := c
	out.APIURL = p.APIURL
	out.APIKey = p.APIKey
	return out, true
}

// Resolve applies environment-variable overrides and built-in defaults.
// TWIG_ADDR overrides APIURL; TWIG_API_KEY overrides APIKey.
// APIURL defaults to "http://localhost:8080" when both env and config are empty.
func (c Config) Resolve() Config {
	if env := os.Getenv("TWIG_ADDR"); env != "" {
		c.APIURL = env
	}
	if env := os.Getenv("TWIG_API_KEY"); env != "" {
		c.APIKey = env
	}
	if c.APIURL == "" {
		c.APIURL = "http://localhost:8080"
	}
	return c
}
