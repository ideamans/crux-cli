package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirPriority(t *testing.T) {
	// CRUX_CLI_HOME wins over everything.
	t.Setenv("CRUX_CLI_HOME", "/tmp/crux-home")
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	if got := Dir(); got != "/tmp/crux-home" {
		t.Fatalf("CRUX_CLI_HOME: got %q, want /tmp/crux-home", got)
	}

	// XDG_CONFIG_HOME is next.
	t.Setenv("CRUX_CLI_HOME", "")
	if got := Dir(); got != filepath.Join("/tmp/xdg", appDirName) {
		t.Fatalf("XDG_CONFIG_HOME: got %q", got)
	}

	// Default is ~/.config/crux-cli.
	t.Setenv("XDG_CONFIG_HOME", "")
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", appDirName)
	if got := Dir(); got != want {
		t.Fatalf("default: got %q, want %q", got, want)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CRUX_CLI_HOME", dir)

	in := Default()
	in.ProjectID = "my-project"
	in.APIKey = "secret"
	if err := Save(in); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Config file must be created at the resolved path with 0600 perms.
	info, err := os.Stat(Path())
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config perm = %o, want 600", perm)
	}

	out, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out.ProjectID != "my-project" || out.APIKey != "secret" {
		t.Errorf("round trip mismatch: %+v", out)
	}
}

func TestCacheDirResolvedDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CRUX_CLI_HOME", dir)
	t.Setenv("CRUX_CACHE_DIR", "")

	c := Default()
	if got, want := c.CacheDirResolved(), filepath.Join(dir, "cache"); got != want {
		t.Errorf("CacheDirResolved = %q, want %q", got, want)
	}

	// config value wins over the default.
	c.CacheDir = "/custom/cache"
	if got := c.CacheDirResolved(); got != "/custom/cache" {
		t.Errorf("CacheDirResolved with config = %q", got)
	}
}
