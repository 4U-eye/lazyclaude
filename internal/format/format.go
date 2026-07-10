// Package format renders tokens, ages and paths for display.
package format

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Tokens renders a token count compactly (828, 96k, 11.7M).
func Tokens(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.0fk", float64(n)/1_000)
	default:
		return strconv.FormatInt(n, 10)
	}
}

// Age renders elapsed time since a unix-milliseconds timestamp (3s, 2m, 15h, 1d).
func Age(epochMs int64) string {
	return AgeAt(epochMs, time.Now())
}

// AgeAt is Age with an injectable clock for tests.
func AgeAt(epochMs int64, now time.Time) string {
	if epochMs == 0 {
		return "-"
	}
	sec := now.Unix() - epochMs/1000
	if sec < 0 {
		sec = 0
	}
	switch {
	case sec < 60:
		return fmt.Sprintf("%ds", sec)
	case sec < 3600:
		return fmt.Sprintf("%dm", sec/60)
	case sec < 86400:
		return fmt.Sprintf("%dh", sec/3600)
	default:
		return fmt.Sprintf("%dd", sec/86400)
	}
}

// ShortCWD abbreviates the home directory to ~ and truncates from the left.
func ShortCWD(cwd string, width int) string {
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(cwd, home) {
		cwd = "~" + cwd[len(home):]
	}
	r := []rune(cwd)
	if len(r) > width && width > 1 {
		return "…" + string(r[len(r)-(width-1):])
	}
	return cwd
}

// ShortModel drops the "claude-" prefix (claude-fable-5 → fable-5).
func ShortModel(model string) string {
	if model == "" {
		return "-"
	}
	return strings.TrimPrefix(model, "claude-")
}
