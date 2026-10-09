package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func clearEnv(t *testing.T) {
	t.Helper()
	t.Setenv("LAZYCLAUDE_NEW_DIR", "")
	t.Setenv("LAZYCLAUDE_CLAUDE_SESSION", "")
	t.Setenv("LAZYCLAUDE_CLAUDE_COMMAND", "")
}

// wantDefault is Default() after the same normalization loadFrom applies
// (home expansion, inherited tmux session, fallback colors).
func wantDefault() Config {
	c := Default()
	normalizeProfiles(&c)
	return c
}

func TestLoadMissingFileUsesDefaults(t *testing.T) {
	clearEnv(t)
	cfg := loadFrom(filepath.Join(t.TempDir(), "nope.toml"))
	if !reflect.DeepEqual(cfg, wantDefault()) {
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
	// claude_session は各プロファイルの tmux セッションに継承される
	for _, p := range cfg.Profiles {
		if p.TmuxSession != "work" {
			t.Errorf("profile %q tmux session = %q, want work", p.Key, p.TmuxSession)
		}
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

	if cfg := loadFrom(path); !reflect.DeepEqual(cfg, wantDefault()) {
		t.Errorf("broken file should fall back to defaults, got %+v", cfg)
	}
}

func TestDefaultProfiles(t *testing.T) {
	clearEnv(t)
	cfg := loadFrom(filepath.Join(t.TempDir(), "nope.toml"))
	if len(cfg.Profiles) != 2 {
		t.Fatalf("want max + enterprise profiles, got %+v", cfg.Profiles)
	}
	home, _ := os.UserHomeDir()
	for i, want := range []struct{ key, label, command, dir string }{
		{"max", "MAX", "claude", filepath.Join(home, ".claude")},
		{"ent", "ENT", "claude-e", filepath.Join(home, ".claude-enterprise")},
	} {
		got := cfg.Profiles[i]
		if got.Key != want.key || got.Label != want.label ||
			got.Command != want.command || got.ConfigDir != want.dir {
			t.Errorf("profile %d = %+v, want %+v", i, got, want)
		}
		if got.Color == "" {
			t.Errorf("profile %d has no color", i)
		}
	}
}

func TestProfilesFromFileReplaceDefaults(t *testing.T) {
	clearEnv(t)
	path := filepath.Join(t.TempDir(), "config.toml")
	os.WriteFile(path, []byte(`
claude_command = "claude --verbose"

[[profiles]]
label = "Work"
config_dir = "~/.claude"

[[profiles]]
key = "ent"
config_dir = "/opt/claude-ent"
command = "claude-e"
tmux_session = "ent"
color = "208"

[[profiles]]
label = "broken"
`), 0o644)

	cfg := loadFrom(path)
	if len(cfg.Profiles) != 2 {
		t.Fatalf("config_dir のないエントリは落とすはず: %+v", cfg.Profiles)
	}
	first := cfg.Profiles[0]
	home, _ := os.UserHomeDir()
	if first.Key != "work" || first.Label != "Work" || first.ConfigDir != filepath.Join(home, ".claude") {
		t.Errorf("first profile = %+v", first)
	}
	// command 未指定の先頭プロファイルは claude_command を引き継ぐ（旧設定の互換）
	if first.Command != "claude --verbose" {
		t.Errorf("first command = %q, want claude_command", first.Command)
	}
	if first.TmuxSession != "claude" {
		t.Errorf("first tmux session = %q, want default", first.TmuxSession)
	}
	second := cfg.Profiles[1]
	if second.Label != "ENT" || second.Command != "claude-e" || second.TmuxSession != "ent" || second.Color != "208" {
		t.Errorf("second profile = %+v", second)
	}
}

func TestClaudeProfiles(t *testing.T) {
	cfg := Config{Profiles: []Profile{{Key: "ent", ConfigDir: "/opt/claude-ent"}}}
	got := cfg.ClaudeProfiles()
	if len(got) != 1 {
		t.Fatalf("got %+v", got)
	}
	if got[0].Key != "ent" ||
		got[0].SessionsDir != "/opt/claude-ent/sessions" ||
		got[0].ProjectsDir != "/opt/claude-ent/projects" {
		t.Errorf("got %+v", got[0])
	}
}

func TestPathHonorsXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")
	if got := Path(); got != "/tmp/xdg-test/lazyclaude/config.toml" {
		t.Errorf("Path() = %q", got)
	}
}
