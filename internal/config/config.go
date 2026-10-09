// Package config loads user configuration.
//
// Precedence: environment variables > config file > built-in defaults.
// The config file lives at $XDG_CONFIG_HOME/lazyclaude/config.toml
// (~/.config/lazyclaude/config.toml by default).
package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/4U-eye/lazyclaude/internal/claude"
)

// Profile is one Claude Code installation to monitor. Claude Code keeps its
// state under CLAUDE_CONFIG_DIR, so a personal plan and an enterprise plan can
// coexist on one machine in separate directories.
type Profile struct {
	// Key is a stable identifier used internally (e.g. "max", "ent").
	Key string `toml:"key"`
	// Label is the short uppercase name shown in the UI (e.g. "MAX", "ENT").
	Label string `toml:"label"`
	// ConfigDir is the installation's CLAUDE_CONFIG_DIR ("~" is expanded).
	ConfigDir string `toml:"config_dir"`
	// Command is typed into a new pane via the interactive shell, so shell
	// aliases and functions apply (e.g. "claude-e").
	Command string `toml:"command"`
	// TmuxSession is the tmux session that hosts sessions created for this
	// profile; empty falls back to ClaudeSession.
	TmuxSession string `toml:"tmux_session"`
	// Color is the profile's accent, as a 256-color index or hex ("#ff8700").
	Color string `toml:"color"`
}

// Config holds all user-tunable settings.
type Config struct {
	// NewDir is the default working directory when creating a session (n key).
	NewDir string `toml:"new_dir"`
	// ClaudeSession is the default tmux session hosting newly created sessions.
	ClaudeSession string `toml:"claude_session"`
	// ClaudeCommand is the launch command for the first profile when the
	// profile itself does not set one (kept for pre-profiles configs).
	ClaudeCommand string `toml:"claude_command"`
	// Profiles are the installations to monitor, in display order.
	Profiles []Profile `toml:"profiles"`
}

// Default returns the built-in settings: the stock ~/.claude installation plus
// the enterprise one at ~/.claude-enterprise (absent directories are skipped
// at scan time, so listing a profile the user lacks costs nothing).
func Default() Config {
	return Config{
		NewDir:        "~",
		ClaudeSession: "claude",
		ClaudeCommand: "claude",
		Profiles: []Profile{
			{Key: "max", Label: "MAX", ConfigDir: "~/.claude", Command: "claude", Color: "45"},
			{Key: "ent", Label: "ENT", ConfigDir: "~/.claude-enterprise", Command: "claude-e", Color: "208"},
		},
	}
}

// Path returns the config file location, honoring XDG_CONFIG_HOME.
func Path() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "lazyclaude", "config.toml")
}

// Load reads the config file (missing or broken files fall back to defaults)
// and applies environment variable overrides.
func Load() Config {
	return loadFrom(Path())
}

func loadFrom(path string) Config {
	cfg := Default()
	// キー単位でマージする。デコード先に直接読ませると、未指定キーが
	// ゼロ値で潰れるかどうかがライブラリの実装依存になるため。
	var file Config
	if _, err := toml.DecodeFile(path, &file); err == nil {
		if file.NewDir != "" {
			cfg.NewDir = file.NewDir
		}
		if file.ClaudeSession != "" {
			cfg.ClaudeSession = file.ClaudeSession
		}
		if file.ClaudeCommand != "" {
			cfg.ClaudeCommand = file.ClaudeCommand
		}
		if len(file.Profiles) > 0 {
			cfg.Profiles = file.Profiles
		}
	}
	applyEnv(&cfg)
	normalizeProfiles(&cfg)
	return cfg
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("LAZYCLAUDE_NEW_DIR"); v != "" {
		cfg.NewDir = v
	}
	if v := os.Getenv("LAZYCLAUDE_CLAUDE_SESSION"); v != "" {
		cfg.ClaudeSession = v
	}
	if v := os.Getenv("LAZYCLAUDE_CLAUDE_COMMAND"); v != "" {
		cfg.ClaudeCommand = v
	}
}

// fallbackColors are handed out to profiles that do not name one, so a
// user-added third installation still renders distinctly.
var fallbackColors = []string{"45", "208", "120", "213", "227", "159"}

// normalizeProfiles fills in defaults and drops entries with no config dir.
// ClaudeCommand applies to the first profile so that a config written before
// profiles existed keeps launching the command it named.
func normalizeProfiles(cfg *Config) {
	out := make([]Profile, 0, len(cfg.Profiles))
	for i, p := range cfg.Profiles {
		p.ConfigDir = ExpandHome(strings.TrimSpace(p.ConfigDir))
		if p.ConfigDir == "" {
			continue
		}
		if p.Key == "" {
			p.Key = strings.ToLower(p.Label)
		}
		if p.Key == "" {
			p.Key = filepath.Base(p.ConfigDir)
		}
		if p.Label == "" {
			p.Label = strings.ToUpper(p.Key)
		}
		if p.Command == "" {
			if i == 0 {
				p.Command = cfg.ClaudeCommand
			} else {
				p.Command = "claude"
			}
		}
		if p.TmuxSession == "" {
			p.TmuxSession = cfg.ClaudeSession
		}
		if p.Color == "" {
			p.Color = fallbackColors[len(out)%len(fallbackColors)]
		}
		out = append(out, p)
	}
	cfg.Profiles = out
}

// ClaudeProfiles converts the configured installations into scan targets.
func (c Config) ClaudeProfiles() []claude.Profile {
	out := make([]claude.Profile, 0, len(c.Profiles))
	for _, p := range c.Profiles {
		out = append(out, claude.NewProfile(p.Key, p.ConfigDir))
	}
	return out
}

// ExpandHome resolves a leading "~" against the user's home directory.
func ExpandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}
