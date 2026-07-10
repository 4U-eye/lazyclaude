// lazyclaude is a lazygit-style TUI for monitoring local Claude Code sessions.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/4U-eye/lazyclaude/internal/claude"
	"github.com/4U-eye/lazyclaude/internal/format"
	"github.com/4U-eye/lazyclaude/internal/ui"
)

func main() {
	list := flag.Bool("list", false, "print sessions as plain text and exit")
	flag.Parse()

	if *list {
		runList()
		return
	}
	p := tea.NewProgram(ui.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func runList() {
	st := claude.DefaultStore()
	sessions := st.Load()
	if len(sessions) == 0 {
		fmt.Println("no running Claude Code sessions")
		return
	}
	seen := claude.LoadSeen(claude.DefaultSeenPath())

	fmt.Printf("%-2s%6s  %-8s%-10s%6s  %6s  %7s  %4s  %-22s  %s\n",
		"U", "PID", "STATUS", "MODEL", "CTX", "OUT", "TOTAL", "AGE", "CWD", "NAME")
	for _, s := range sessions {
		tail := claude.TailFromFile(s.Transcript, 64*1024)
		stats := st.ScanTranscript(s.Transcript)

		unread := " "
		if seen.IsUnread(s) {
			unread = "!"
		}
		ctx := "-"
		if tail.HasUsage {
			ctx = format.Tokens(tail.ContextTokens)
		}
		status := s.Status
		if status == "" {
			status = "?"
		}
		name := stats.Title
		if name == "" {
			name = "-"
		}
		fmt.Printf("%-2s%6d  %-8s%-10s%6s  %6s  %7s  %4s  %-22s  %s\n",
			unread, s.PID, status, format.ShortModel(tail.Model),
			ctx, format.Tokens(stats.Output), format.Tokens(stats.Total()),
			format.Age(s.UpdatedAt), format.ShortCWD(s.CWD, 22), name)
	}
}
