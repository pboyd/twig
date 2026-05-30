package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	APIURL   string         `toml:"api_url"`
	APIKey   string         `toml:"api_key"`
	Pomodoro PomodoroConfig `toml:"pomodoro"`
}

type PomodoroConfig struct {
	OnStart    string `toml:"on_start"`
	OnCancel   string `toml:"on_cancel"`
	OnComplete string `toml:"on_complete"`
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
