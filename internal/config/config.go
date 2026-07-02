package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	appDirName   = "crux-cli"
	fileName     = "config.json"
	envConfigDir = "CRUX_CLI_HOME"
)

type Config struct {
	ProjectID     string `json:"project_id"`
	APIKey        string `json:"api_key"`
	CacheDir      string `json:"cache_dir"`
	DefaultFormat string `json:"default_format"`
	DefaultMonths int    `json:"default_months"`
}

func Default() *Config {
	return &Config{
		DefaultFormat: "table",
		DefaultMonths: 12,
	}
}

// Dir returns the config directory.
// Priority: CRUX_CLI_HOME > $XDG_CONFIG_HOME/crux-cli > ~/.config/crux-cli.
// On macOS this uses ~/.config (not ~/Library) to match the CLI convention.
func Dir() string {
	if d := os.Getenv(envConfigDir); d != "" {
		return d
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, appDirName)
}

func Path() string {
	return filepath.Join(Dir(), fileName)
}

// legacyPath is the pre-v0.4 location (~/.crux-cli/config.json). Load falls
// back to it so existing users keep their settings; the next Save migrates
// the file to the new location under ~/.config.
func legacyPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".crux-cli", "config.json")
}

func Load() (*Config, error) {
	cfg := Default()
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			// Fall back to the legacy location if present.
			if legacy, lerr := os.ReadFile(legacyPath()); lerr == nil {
				if uerr := json.Unmarshal(legacy, cfg); uerr != nil {
					return nil, uerr
				}
				return cfg, nil
			}
			return cfg, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func Save(cfg *Config) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), data, 0o600)
}

// CacheDirResolved returns the effective cache directory.
// Priority: config file > CRUX_CACHE_DIR env > default (<config dir>/cache).
func (c *Config) CacheDirResolved() string {
	if c.CacheDir != "" {
		return c.CacheDir
	}
	if env := os.Getenv("CRUX_CACHE_DIR"); env != "" {
		return env
	}
	return filepath.Join(Dir(), "cache")
}

// ProjectIDResolved returns the effective project ID.
// Priority: flag value > CRUX_PROJECT env > config file.
func (c *Config) ProjectIDResolved(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if env := os.Getenv("CRUX_PROJECT"); env != "" {
		return env
	}
	return c.ProjectID
}

// APIKeyResolved returns the effective CrUX API key.
// Priority: flag value > CRUX_API_KEY env > config file.
func (c *Config) APIKeyResolved(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if env := os.Getenv("CRUX_API_KEY"); env != "" {
		return env
	}
	return c.APIKey
}
