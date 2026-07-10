package format

import (
	"testing"
	"time"
)

func TestTokens(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{828, "828"},
		{999, "999"},
		{1_000, "1k"},
		{96_400, "96k"},
		{999_499, "999k"},
		{1_000_000, "1.0M"},
		{11_700_000, "11.7M"},
	}
	for _, c := range cases {
		if got := Tokens(c.n); got != c.want {
			t.Errorf("Tokens(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestAgeAt(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	ms := func(secAgo int64) int64 { return (1_000_000 - secAgo) * 1000 }
	cases := []struct {
		epochMs int64
		want    string
	}{
		{0, "-"},
		{ms(5), "5s"},
		{ms(90), "1m"},
		{ms(3 * 3600), "3h"},
		{ms(2 * 86400), "2d"},
		{ms(-10), "0s"}, // 未来のタイムスタンプは0s扱い
	}
	for _, c := range cases {
		if got := AgeAt(c.epochMs, now); got != c.want {
			t.Errorf("AgeAt(%d) = %q, want %q", c.epochMs, got, c.want)
		}
	}
}

func TestShortModel(t *testing.T) {
	if got := ShortModel("claude-fable-5"); got != "fable-5" {
		t.Errorf("got %q", got)
	}
	if got := ShortModel(""); got != "-" {
		t.Errorf("got %q", got)
	}
	if got := ShortModel("gpt-x"); got != "gpt-x" {
		t.Errorf("got %q", got)
	}
}

func TestShortCWD(t *testing.T) {
	if got := ShortCWD("/opt/some/dir", 20); got != "/opt/some/dir" {
		t.Errorf("got %q", got)
	}
	got := ShortCWD("/opt/very/long/path/to/some/project", 10)
	if len([]rune(got)) != 10 || got[:len("…")] != "…" {
		t.Errorf("truncated form should be 10 runes starting with …, got %q", got)
	}
}
