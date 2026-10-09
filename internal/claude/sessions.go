// Package claude reads Claude Code's local session registry and transcripts.
//
// Data sources (undocumented internal format of Claude Code; parsed defensively):
//   - {configDir}/sessions/{pid}.json                       live session registry
//   - {configDir}/projects/{escaped-cwd}/{sessionId}.jsonl  session transcript
//
// configDir is ~/.claude by default, but Claude Code honors CLAUDE_CONFIG_DIR,
// so one machine can host several installations side by side (e.g. a personal
// plan in ~/.claude and an enterprise one in ~/.claude-enterprise). A Store
// therefore scans a list of profiles rather than a single directory.
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
	Profile    string `json:"-"` // key of the profile this session was found in
}

// Profile is one Claude Code installation (one CLAUDE_CONFIG_DIR).
type Profile struct {
	Key         string // stable identifier, e.g. "max" / "ent"
	SessionsDir string
	ProjectsDir string
}

// NewProfile derives the registry and transcript directories from a config dir.
func NewProfile(key, configDir string) Profile {
	return Profile{
		Key:         key,
		SessionsDir: filepath.Join(configDir, "sessions"),
		ProjectsDir: filepath.Join(configDir, "projects"),
	}
}

// Store scans one or more Claude Code installations and caches transcript scans.
type Store struct {
	Profiles []Profile

	scanMu    sync.Mutex
	scanCache map[string]scanEntry
}

// DefaultStore points at the stock ~/.claude installation only.
// Callers that support multiple plans should build a Store from config.
func DefaultStore() *Store {
	home, _ := os.UserHomeDir()
	return &Store{Profiles: []Profile{NewProfile("max", filepath.Join(home, ".claude"))}}
}

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]`)

// projectDir maps a session cwd to its transcript directory
// (Claude Code replaces every non-alphanumeric rune with "-").
func (p Profile) projectDir(cwd string) string {
	return filepath.Join(p.ProjectsDir, nonAlnum.ReplaceAllString(cwd, "-"))
}

var statusOrder = map[string]int{"busy": 0, "waiting": 1, "idle": 2}

// Load returns sessions from every profile whose process is still alive,
// sorted by status (busy → waiting → idle) then recency.
func (st *Store) Load() []Session {
	var sessions []Session
	for _, p := range st.Profiles {
		sessions = append(sessions, p.load()...)
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

// load reads one profile's registry. A missing directory yields no sessions,
// so an installation the user does not have is simply absent.
func (p Profile) load() []Session {
	entries, err := os.ReadDir(p.SessionsDir)
	if err != nil {
		return nil
	}
	var sessions []Session
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(p.SessionsDir, e.Name()))
		if err != nil {
			continue
		}
		var s Session
		if json.Unmarshal(raw, &s) != nil || s.PID <= 0 || !pidAlive(s.PID) {
			continue
		}
		s.Transcript = filepath.Join(p.projectDir(s.CWD), s.SessionID+".jsonl")
		s.Profile = p.Key
		sessions = append(sessions, s)
	}
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
