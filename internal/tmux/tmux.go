// Package tmux bridges lazyclaude to tmux: pane discovery for Claude Code
// processes, live screen capture, key injection, and session lifecycle.
package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Pane identifies a tmux pane hosting a Claude Code session.
type Pane struct {
	ID   string // "%42"
	Name string // "work:3.1" (session:window.pane)
}

func run(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	out, err := cmd.Output()
	return string(out), err
}

// Available reports whether tmux is installed.
func Available() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// FindPane locates the tmux pane whose tty matches the process's tty.
func FindPane(pid int) *Pane {
	tty, err := run4ps(pid)
	if err != nil || tty == "" || tty == "??" {
		return nil
	}
	out, err := run("list-panes", "-a", "-F",
		"#{pane_tty}|#{pane_id}|#{session_name}:#{window_index}.#{pane_index}")
	if err != nil {
		return nil
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		parts := strings.SplitN(line, "|", 3)
		if len(parts) == 3 && parts[0] == "/dev/"+tty {
			return &Pane{ID: parts[1], Name: parts[2]}
		}
	}
	return nil
}

func run4ps(pid int) (string, error) {
	out, err := exec.Command("ps", "-o", "tty=", "-p", fmt.Sprint(pid)).Output()
	return strings.TrimSpace(string(out)), err
}

// Capture returns the pane's visible screen. With ansi=true, SGR color/attr
// escape sequences are preserved.
func Capture(paneID string, ansi bool) ([]string, error) {
	args := []string{"capture-pane", "-p"}
	if ansi {
		args = append(args, "-e")
	}
	args = append(args, "-t", paneID)
	out, err := run(args...)
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimRight(out, "\n"), "\n"), nil
}

// Send types text into the pane and presses Enter.
// The literal text and the Enter key are sent separately (with a short gap)
// so the target's paste detection doesn't swallow the newline.
func Send(paneID, text string) error {
	if _, err := run("send-keys", "-t", paneID, "-l", text); err != nil {
		return err
	}
	time.Sleep(100 * time.Millisecond)
	_, err := run("send-keys", "-t", paneID, "Enter")
	return err
}

// SelectPane makes the pane active within its session (window + pane).
func SelectPane(paneID string) {
	run("select-window", "-t", paneID) //nolint:errcheck // best effort
	run("select-pane", "-t", paneID)   //nolint:errcheck
}

// InsideTmux reports whether lazyclaude itself runs inside tmux.
func InsideTmux() bool { return os.Getenv("TMUX") != "" }

// SwitchClient moves the current tmux client to the pane's session.
// Only meaningful when running inside tmux.
func SwitchClient(paneID string) error {
	_, err := run("switch-client", "-t", paneID)
	return err
}

// AttachCmd builds the command to attach to the pane's session from outside
// tmux. Run it with the terminal released (tea.ExecProcess).
func AttachCmd(paneID string) *exec.Cmd {
	return exec.Command("tmux", "attach", "-t", paneID)
}

// CreateSession opens a new window (creating the session when missing),
// then launches the command via the interactive shell so user aliases apply.
func CreateSession(sessionName, cwd, command string) (paneID string, err error) {
	cwd = expandHome(cwd)
	if fi, err := os.Stat(cwd); err != nil || !fi.IsDir() {
		return "", fmt.Errorf("no such directory: %s", cwd)
	}
	var out string
	if _, err := run("has-session", "-t", "="+sessionName); err == nil {
		out, err = run("new-window", "-t", sessionName, "-c", cwd, "-P", "-F", "#{pane_id}")
		if err != nil {
			return "", fmt.Errorf("new-window failed: %w", err)
		}
	} else {
		out, err = run("new-session", "-d", "-s", sessionName, "-c", cwd, "-P", "-F", "#{pane_id}")
		if err != nil {
			return "", fmt.Errorf("new-session failed: %w", err)
		}
	}
	paneID = strings.TrimSpace(out)
	if _, err := run("send-keys", "-t", paneID, "-l", command); err != nil {
		return paneID, err
	}
	time.Sleep(100 * time.Millisecond)
	if _, err := run("send-keys", "-t", paneID, "Enter"); err != nil {
		return paneID, err
	}
	return paneID, nil
}

// KillPane terminates the pane (and its process).
func KillPane(paneID string) error {
	_, err := run("kill-pane", "-t", paneID)
	return err
}

func expandHome(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}
