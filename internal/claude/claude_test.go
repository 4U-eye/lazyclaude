package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// writeFile is a test helper; fixture contents are dummy data only.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	return &Store{
		SessionsDir: filepath.Join(dir, "sessions"),
		ProjectsDir: filepath.Join(dir, "projects"),
	}
}

func registryJSON(pid int, sid, cwd, status string, updatedAt int64) string {
	b, _ := json.Marshal(map[string]any{
		"pid": pid, "sessionId": sid, "cwd": cwd,
		"status": status, "updatedAt": updatedAt, "version": "0.0.0",
	})
	return string(b)
}

func TestLoadFiltersDeadAndSorts(t *testing.T) {
	st := newTestStore(t)
	alive := os.Getpid()
	const dead = 999_999_999 // まず存在しないpid

	writeFile(t, filepath.Join(st.SessionsDir, "1.json"),
		registryJSON(alive, "sid-idle", "/tmp/proj-a", "idle", 100))
	writeFile(t, filepath.Join(st.SessionsDir, "2.json"),
		registryJSON(alive, "sid-busy", "/tmp/proj-b", "busy", 50))
	writeFile(t, filepath.Join(st.SessionsDir, "3.json"),
		registryJSON(dead, "sid-dead", "/tmp/proj-c", "busy", 999))
	writeFile(t, filepath.Join(st.SessionsDir, "4.json"),
		registryJSON(alive, "sid-wait", "/tmp/proj-d", "waiting", 10))
	writeFile(t, filepath.Join(st.SessionsDir, "broken.json"), "{not json")

	got := st.Load()
	if len(got) != 3 {
		t.Fatalf("want 3 sessions, got %d", len(got))
	}
	order := []string{got[0].SessionID, got[1].SessionID, got[2].SessionID}
	want := []string{"sid-busy", "sid-wait", "sid-idle"}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("sort order = %v, want %v", order, want)
		}
	}
	// transcriptパス: cwdの非英数字が-に変換される
	wantPath := filepath.Join(st.ProjectsDir, "-tmp-proj-b", "sid-busy.jsonl")
	if got[0].Transcript != wantPath {
		t.Errorf("transcript = %q, want %q", got[0].Transcript, wantPath)
	}
}

func usageLine(in, out, cacheW, cacheR int64) string {
	return fmt.Sprintf(
		`{"type":"assistant","message":{"model":"claude-dummy-1","usage":{"input_tokens":%d,"output_tokens":%d,"cache_creation_input_tokens":%d,"cache_read_input_tokens":%d}}}`,
		in, out, cacheW, cacheR)
}

func TestScanTranscript(t *testing.T) {
	st := newTestStore(t)
	path := filepath.Join(t.TempDir(), "sample.jsonl")
	writeFile(t, path, ""+
		`{"type":"ai-title","aiTitle":"Old title"}`+"\n"+
		usageLine(10, 100, 1000, 10000)+"\n"+
		`{"type":"user","message":{"content":"dummy prompt"}}`+"\n"+
		usageLine(5, 200, 0, 20000)+"\n"+
		`{"type":"ai-title","aiTitle":"Sample refactoring session"}`+"\n"+
		`{"type":"assistant","message":{"usage":{"input_tokens":7}}}`+"\n"+ // output_tokensなし → 集計対象外
		"{broken json\n")

	stats := st.ScanTranscript(path)
	if stats.Input != 15 || stats.Output != 300 || stats.CacheW != 1000 || stats.CacheR != 30000 {
		t.Fatalf("sums = %+v", stats)
	}
	if stats.Count != 2 {
		t.Errorf("count = %d, want 2", stats.Count)
	}
	if stats.Title != "Sample refactoring session" {
		t.Errorf("title = %q (最後のai-titleを採用するはず)", stats.Title)
	}
	if stats.Total() != 15+300+1000+30000 {
		t.Errorf("total = %d", stats.Total())
	}

	// キャッシュ: 同サイズなら再走査しない（ファイル差し替えでも同サイズなら旧値）
	stats2 := st.ScanTranscript(path)
	if stats2 != stats {
		t.Errorf("cached result differs: %+v vs %+v", stats2, stats)
	}

	// ファイルが成長したら再計算される
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	fmt.Fprintln(f, usageLine(1, 1, 0, 0))
	f.Close()
	stats3 := st.ScanTranscript(path)
	if stats3.Output != 301 || stats3.Count != 3 {
		t.Errorf("after growth: %+v", stats3)
	}
}

func TestScanTranscriptMissing(t *testing.T) {
	st := newTestStore(t)
	if got := st.ScanTranscript("/no/such/file.jsonl"); got != (Stats{}) {
		t.Errorf("missing file should yield zero stats, got %+v", got)
	}
}

func TestTailFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tail.jsonl")
	writeFile(t, path, ""+
		usageLine(100, 1, 0, 0)+"\n"+
		`{"type":"assistant","message":{"model":"claude-dummy-2","usage":{"input_tokens":3,"output_tokens":9,"cache_read_input_tokens":200,"cache_creation_input_tokens":40}}}`+"\n")

	info := TailFromFile(path, DefaultTailBytes)
	if !info.HasUsage {
		t.Fatal("HasUsage should be true")
	}
	if info.ContextTokens != 3+200+40 {
		t.Errorf("context = %d, want 243", info.ContextTokens)
	}
	if info.Model != "claude-dummy-2" {
		t.Errorf("model = %q", info.Model)
	}

	if got := TailFromFile("/no/such/file", DefaultTailBytes); got.HasUsage || got.Model != "" {
		t.Errorf("missing file should yield empty info, got %+v", got)
	}
}

func TestSeen(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state", "seen.json")

	// 保存 → 読み込みの往復
	s := Seen{"sid-a": 100}
	if err := s.Save(statePath); err != nil {
		t.Fatal(err)
	}
	loaded := LoadSeen(statePath)
	if loaded["sid-a"] != 100 {
		t.Fatalf("roundtrip failed: %+v", loaded)
	}

	// 壊れたファイル・欠損ファイルは空mapに
	writeFile(t, filepath.Join(dir, "broken.json"), "not json")
	if got := LoadSeen(filepath.Join(dir, "broken.json")); len(got) != 0 {
		t.Errorf("broken file should yield empty seen")
	}
	if got := LoadSeen(filepath.Join(dir, "missing.json")); len(got) != 0 {
		t.Errorf("missing file should yield empty seen")
	}

	// 未読判定: 成長したら未読 / 同サイズ既読 / 未知セッションは既読扱い
	tr := filepath.Join(dir, "t.jsonl")
	writeFile(t, tr, "0123456789") // 10 bytes
	sess := Session{SessionID: "sid-a", Transcript: tr}
	if !(Seen{"sid-a": 5}).IsUnread(sess) {
		t.Error("grown transcript should be unread")
	}
	if (Seen{"sid-a": 10}).IsUnread(sess) {
		t.Error("same size should be read")
	}
	if (Seen{}).IsUnread(sess) {
		t.Error("unknown session should count as read")
	}

	// MarkSeen で現サイズが記録される
	m := Seen{}
	m.MarkSeen(sess)
	if m["sid-a"] != 10 {
		t.Errorf("MarkSeen recorded %d, want 10", m["sid-a"])
	}
}
