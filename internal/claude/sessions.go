// Package claude reads Claude Code's local session registry and transcripts.
//
// Data sources (undocumented internal format of Claude Code; parsed defensively):
//   - ~/.claude/sessions/{pid}.json                       live session registry
//   - ~/.claude/projects/{escaped-cwd}/{sessionId}.jsonl  session transcript
package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
)

// Session is one live Claude Code session from the registry.
type Session struct {
	PID       int    `json:"pid"`
	SessionID string `json:"sessionId"`
	CWD       string `json:"cwd"`
	Status    string `json:"status"` // "busy" | "waiting" | "idle" (observed values)
	UpdatedAt int64  `json:"updatedAt"`
	Version   string `json:"version"`
	Kind      string `json:"kind"`

	Transcript string `json:"-"` // resolved transcript path
}

// Store locates Claude Code's data directories and caches transcript scans.
type Store struct {
	SessionsDir string
	ProjectsDir string

	scanMu    sync.Mutex
	scanCache map[string]scanEntry
}

// DefaultStore points at the real ~/.claude directories.
func DefaultStore() *Store {
	home, _ := os.UserHomeDir()
	return &Store{
		SessionsDir: filepath.Join(home, ".claude", "sessions"),
		ProjectsDir: filepath.Join(home, ".claude", "projects"),
	}
}

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]`)

// projectDir maps a session cwd to its transcript directory
// (Claude Code replaces every non-alphanumeric rune with "-").
func (st *Store) projectDir(cwd string) string {
	return filepath.Join(st.ProjectsDir, nonAlnum.ReplaceAllString(cwd, "-"))
}

var statusOrder = map[string]int{"busy": 0, "waiting": 1, "idle": 2}

// Load returns sessions whose process is still alive,
// sorted by status (busy → waiting → idle) then recency.
func (st *Store) Load() []Session {
	entries, err := os.ReadDir(st.SessionsDir)
	if err != nil {
		return nil
	}
	var sessions []Session
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(st.SessionsDir, e.Name()))
		if err != nil {
			continue
		}
		var s Session
		if json.Unmarshal(raw, &s) != nil || s.PID <= 0 || !pidAlive(s.PID) {
			continue
		}
		s.Transcript = filepath.Join(st.projectDir(s.CWD), s.SessionID+".jsonl")
		sessions = append(sessions, s)
	}
	sort.SliceStable(sessions, func(i, j int) bool {
		oi, ok := statusOrder[sessions[i].Status]
		if !ok {
			oi = 9
		}
		oj, ok := statusOrder[sessions[j].Status]
		if !ok {
			oj = 9
		}
		if oi != oj {
			return oi < oj
		}
		return sessions[i].UpdatedAt > sessions[j].UpdatedAt
	})
	return sessions
}

// pidAlive reports whether a process with the given pid exists.
// EPERM still means the process exists (owned by someone else).
func pidAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// TranscriptSize returns the transcript file size, or 0 when missing.
func TranscriptSize(s Session) int64 {
	fi, err := os.Stat(s.Transcript)
	if err != nil {
		return 0
	}
	return fi.Size()
}
