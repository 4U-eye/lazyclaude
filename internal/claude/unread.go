package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Seen maps sessionId to the transcript size at the moment the user last
// opened that session. A transcript larger than its recorded size is unread.
type Seen map[string]int64

// DefaultSeenPath is where read/unread state persists across runs.
func DefaultSeenPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "lazyclaude", "seen.json")
}

// LoadSeen reads persisted state; missing or broken files yield an empty map.
func LoadSeen(path string) Seen {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Seen{}
	}
	var s Seen
	if json.Unmarshal(raw, &s) != nil || s == nil {
		return Seen{}
	}
	return s
}

// Save persists the state, creating parent directories as needed.
func (s Seen) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// IsUnread reports whether the transcript grew past the last-seen size.
// Sessions never seen before count as read (avoids all-unread noise on first run).
func (s Seen) IsUnread(sess Session) bool {
	last, ok := s[sess.SessionID]
	return ok && TranscriptSize(sess) > last
}

// MarkSeen records the current transcript size as read.
func (s Seen) MarkSeen(sess Session) {
	if sess.SessionID != "" {
		s[sess.SessionID] = TranscriptSize(sess)
	}
}
