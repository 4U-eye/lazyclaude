// Package config loads user configuration.
//
// Precedence: environment variables > config file > built-in defaults.
// The config file lives at $XDG_CONFIG_HOME/lazyclaude/config.toml
// (~/.config/lazyclaude/config.toml by default).
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds all user-tunable settings.
type Config struct {
	// NewDir is the default working directory when creating a session (n key).
	NewDir string `toml:"new_dir"`
	// ClaudeSession is the tmux session that hosts newly created sessions.
	ClaudeSession string `toml:"claude_session"`
	// ClaudeCommand is typed into the new pane via the interactive shell,
	// so shell aliases apply.
	ClaudeCommand string `toml:"claude_command"`
}

// Default returns the built-in settings.
func Default() Config {
	return Config{
		NewDir:        "~",
		ClaudeSession: "claude",
		ClaudeCommand: "claude",
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
	toml.DecodeFile(path, &cfg) //nolint:errcheck // 欠損・破損はデフォルトで続行
	applyEnv(&cfg)
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
