package tmux

import (
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

// These are integration tests against a real tmux server, using throwaway
// sessions only. They skip when tmux is unavailable.

func requireTmux(t *testing.T) {
	t.Helper()
	if !Available() {
		t.Skip("tmux not installed")
	}
}

func killSession(name string) {
	exec.Command("tmux", "kill-session", "-t", name).Run() //nolint:errcheck
}

func TestCreateSessionCaptureSend(t *testing.T) {
	requireTmux(t)
	const name = "lazyclaude-go-test-a"
	killSession(name)
	t.Cleanup(func() { killSession(name) })

	paneID, err := CreateSession(name, "~", "cat")
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if !strings.HasPrefix(paneID, "%") {
		t.Fatalf("pane id = %q", paneID)
	}

	// catが起動するのを待ってから送信 → エコーバックをcaptureで確認
	time.Sleep(500 * time.Millisecond)
	if err := Send(paneID, "hello from go test"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	lines, err := Capture(paneID, false)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "hello from go test") {
		t.Errorf("capture does not contain sent text:\n%s", joined)
	}

	// 2つ目はnew-windowで同一セッションに増える
	pane2, err := CreateSession(name, "~", "cat")
	if err != nil {
		t.Fatalf("second CreateSession: %v", err)
	}
	if pane2 == paneID {
		t.Errorf("second pane should differ: %s", pane2)
	}

	// 不正なディレクトリはエラー
	if _, err := CreateSession(name, "/no/such/dir-xyz", "cat"); err == nil {
		t.Error("invalid dir should error")
	}
}

func TestFindPaneAndKill(t *testing.T) {
	requireTmux(t)
	const name = "lazyclaude-go-test-b"
	killSession(name)
	t.Cleanup(func() { killSession(name) })

	out, err := exec.Command("tmux", "new-session", "-d", "-s", name,
		"-P", "-F", "#{pane_id}|#{pane_pid}", "sleep 60").Output()
	if err != nil {
		t.Fatalf("new-session: %v", err)
	}
	parts := strings.SplitN(strings.TrimSpace(string(out)), "|", 2)
	paneID := parts[0]
	pid, _ := strconv.Atoi(parts[1])

	p := FindPane(pid)
	if p == nil {
		t.Fatalf("FindPane(%d) = nil", pid)
	}
	if p.ID != paneID {
		t.Errorf("pane id = %q, want %q", p.ID, paneID)
	}
	if !strings.HasPrefix(p.Name, name+":") {
		t.Errorf("pane name = %q", p.Name)
	}

	if FindPane(999_999_999) != nil {
		t.Error("bogus pid should not resolve to a pane")
	}

	if err := KillPane(paneID); err != nil {
		t.Fatalf("KillPane: %v", err)
	}
	time.Sleep(300 * time.Millisecond)
	if err := exec.Command("tmux", "has-session", "-t", "="+name).Run(); err == nil {
		t.Error("session should be gone after killing its only pane")
	}
}
