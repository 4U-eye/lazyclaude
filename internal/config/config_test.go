package config

import (
	"os"
	"path/filepath"
	"testing"
)

func clearEnv(t *testing.T) {
	t.Helper()
	t.Setenv("LAZYCLAUDE_NEW_DIR", "")
	t.Setenv("LAZYCLAUDE_CLAUDE_SESSION", "")
	t.Setenv("LAZYCLAUDE_CLAUDE_COMMAND", "")
}

func TestLoadMissingFileUsesDefaults(t *testing.T) {
	clearEnv(t)
	cfg := loadFrom(filepath.Join(t.TempDir(), "nope.toml"))
	if cfg != Default() {
		t.Errorf("got %+v, want defaults", cfg)
	}
}

func TestLoadFromFile(t *testing.T) {
	clearEnv(t)
	path := filepath.Join(t.TempDir(), "config.toml")
	os.WriteFile(path, []byte("new_dir = \"~/projects\"\nclaude_session = \"work\"\n"), 0o644)

	cfg := loadFrom(path)
	if cfg.NewDir != "~/projects" || cfg.ClaudeSession != "work" {
		t.Errorf("got %+v", cfg)
	}
	if cfg.ClaudeCommand != "claude" {
		t.Errorf("unspecified key should keep default, got %q", cfg.ClaudeCommand)
	}
}

func TestEnvOverridesFile(t *testing.T) {
	clearEnv(t)
	path := filepath.Join(t.TempDir(), "config.toml")
	os.WriteFile(path, []byte("new_dir = \"~/projects\"\n"), 0o644)
	t.Setenv("LAZYCLAUDE_NEW_DIR", "~/from-env")

	if cfg := loadFrom(path); cfg.NewDir != "~/from-env" {
		t.Errorf("env should win, got %q", cfg.NewDir)
	}
}

func TestBrokenFileFallsBack(t *testing.T) {
	clearEnv(t)
	path := filepath.Join(t.TempDir(), "config.toml")
	os.WriteFile(path, []byte("{{{not toml"), 0o644)

	if cfg := loadFrom(path); cfg != Default() {
		t.Errorf("broken file should fall back to defaults, got %+v", cfg)
	}
}

func TestPathHonorsXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")
	if got := Path(); got != "/tmp/xdg-test/lazyclaude/config.toml" {
		t.Errorf("Path() = %q", got)
	}
}
