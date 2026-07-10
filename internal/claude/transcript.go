package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
)

// maxLineBytes bounds a single JSONL line; transcripts can embed large blobs.
const maxLineBytes = 16 * 1024 * 1024

// Stats are cumulative token counts and metadata for one transcript.
type Stats struct {
	Input  int64
	Output int64
	CacheW int64 // cache_creation_input_tokens
	CacheR int64 // cache_read_input_tokens
	Count  int   // usage-bearing messages
	Title  string
}

// Total is the overall consumed tokens (input + output + both cache kinds).
func (s Stats) Total() int64 { return s.Input + s.Output + s.CacheW + s.CacheR }

type scanEntry struct {
	size  int64
	stats Stats
}

type usageJSON struct {
	InputTokens              *int64 `json:"input_tokens"`
	OutputTokens             *int64 `json:"output_tokens"`
	CacheCreationInputTokens *int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     *int64 `json:"cache_read_input_tokens"`
}

func deref(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// ScanTranscript walks the whole transcript once, accumulating token usage
// and picking up the session title (last "ai-title" record).
// Results are cached by file size, so unchanged files cost one stat call.
func (st *Store) ScanTranscript(path string) Stats {
	fi, err := os.Stat(path)
	if err != nil {
		return Stats{}
	}
	st.scanMu.Lock()
	if e, ok := st.scanCache[path]; ok && e.size == fi.Size() {
		st.scanMu.Unlock()
		return e.stats
	}
	st.scanMu.Unlock()

	f, err := os.Open(path)
	if err != nil {
		return Stats{}
	}
	defer f.Close()

	var stats Stats
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
	for sc.Scan() {
		line := sc.Bytes()
		if bytes.Contains(line, []byte(`"ai-title"`)) {
			var rec struct {
				AITitle string `json:"aiTitle"`
			}
			if json.Unmarshal(line, &rec) == nil && rec.AITitle != "" {
				stats.Title = rec.AITitle
			}
			continue
		}
		if !bytes.Contains(line, []byte(`"output_tokens"`)) {
			continue
		}
		var rec struct {
			Message struct {
				Usage *usageJSON `json:"usage"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &rec) != nil {
			continue
		}
		u := rec.Message.Usage
		if u == nil || u.OutputTokens == nil {
			continue
		}
		stats.Input += deref(u.InputTokens)
		stats.Output += deref(u.OutputTokens)
		stats.CacheW += deref(u.CacheCreationInputTokens)
		stats.CacheR += deref(u.CacheReadInputTokens)
		stats.Count++
	}

	st.scanMu.Lock()
	if st.scanCache == nil {
		st.scanCache = map[string]scanEntry{}
	}
	st.scanCache[path] = scanEntry{fi.Size(), stats}
	st.scanMu.Unlock()
	return stats
}

// TailInfo is the most recent usage/model state near the end of a transcript.
type TailInfo struct {
	HasUsage      bool
	ContextTokens int64 // latest input + cache_read + cache_creation (current context size)
	Model         string
}

// DefaultTailBytes is how far back TailFromFile looks by default.
const DefaultTailBytes = 256 * 1024

// TailFromFile reads at most maxBytes from the end of the transcript and
// extracts the latest usage (context size) and model.
func TailFromFile(path string, maxBytes int64) TailInfo {
	f, err := os.Open(path)
	if err != nil {
		return TailInfo{}
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return TailInfo{}
	}
	if fi.Size() > maxBytes {
		if _, err := f.Seek(fi.Size()-maxBytes, io.SeekStart); err != nil {
			return TailInfo{}
		}
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
	first := fi.Size() > maxBytes // 途中から読んだ最初の不完全な行を捨てる
	var info TailInfo
	for sc.Scan() {
		if first {
			first = false
			continue
		}
		line := sc.Bytes()
		if !bytes.Contains(line, []byte(`"message"`)) {
			continue
		}
		var rec struct {
			Message struct {
				Model string     `json:"model"`
				Usage *usageJSON `json:"usage"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &rec) != nil {
			continue
		}
		if rec.Message.Model != "" {
			info.Model = rec.Message.Model
		}
		if u := rec.Message.Usage; u != nil {
			info.HasUsage = true
			info.ContextTokens = deref(u.InputTokens) +
				deref(u.CacheReadInputTokens) + deref(u.CacheCreationInputTokens)
		}
	}
	return info
}
