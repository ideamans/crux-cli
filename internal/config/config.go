package config

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".crux-cli")
}

func Path() string {
	return filepath.Join(Dir(), "config.json")
}

func Load() (*Config, error) {
	cfg := Default()
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
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
	if err := os.MkdirAll(Dir(), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), data, 0644)
}

// CacheDirResolved returns the effective cache directory.
// Priority: config file > CRUX_CACHE_DIR env > default (~/.crux-cli/cache).
func (c *Config) CacheDirResolved() string {
	if c.CacheDir != "" {
		return c.CacheDir
	}
	if env := os.Getenv("CRUX_CACHE_DIR"); env != "" {
		return env
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".crux-cli", "cache")
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
