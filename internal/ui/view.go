package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/4U-eye/lazyclaude/internal/claude"
	"github.com/4U-eye/lazyclaude/internal/format"
)

// padANSI truncates (ANSI-aware) and right-pads to exactly w display columns.
func padANSI(s string, w int) string {
	s = ansi.Truncate(s, w, "")
	if gap := w - ansi.StringWidth(s); gap > 0 {
		s += strings.Repeat(" ", gap)
	}
	return s
}

// panelBgSGR is the raw SGR to re-apply the panel background. It must match
// cPanelBg in theme.go. Kept as a string constant so we can splice it into
// content without going through lipgloss (which always emits a full reset
// at the end of each Render() call and would break the outer background).
const panelBgSGR = "\x1b[48;5;235m"

// padPanelLine truncates content to w columns, then re-applies the panel
// background after every inner SGR reset so gaps between styled spans (and
// the right-side padding) are filled instead of falling back to the
// terminal's default background — which shows up as "holes" in the panel.
func padPanelLine(s string, w int) string {
	s = ansi.Truncate(s, w, "")
	s = strings.ReplaceAll(s, "\x1b[0m", "\x1b[0m"+panelBgSGR)
	s = strings.ReplaceAll(s, "\x1b[m", "\x1b[m"+panelBgSGR)
	s = panelBgSGR + s
	if gap := w - ansi.StringWidth(s); gap > 0 {
		s += strings.Repeat(" ", gap)
	}
	return s + "\x1b[0m"
}

func (m Model) View() string {
	if m.width < minWidth || m.height < minHeight {
		return sWait.Render("画面が小さすぎます")
	}

	var body string
	if m.mode == modeViewer || (m.mode == modeInput && m.inputFor == inputInstruction) {
		body = m.viewViewerBody()
	} else {
		body = m.viewListBody()
	}

	var bottom string
	switch m.mode {
	case modeInput:
		bottom = sTitlebar.Width(m.width).Render(m.input.View())
	case modeConfirm:
		bottom = sWarnBar.Width(m.width).Render(" " + m.confirmMsg + "  [y/N]")
	case modeViewer:
		bottom = m.keybar([][2]string{{"i", "指示を送信"}, {"a", "ペインへ移動"}, {"q", "戻る"}})
	default:
		bottom = m.keybar([][2]string{
			{"j/k", "移動"}, {"enter", "開く"}, {"n", "新規"},
			{"x", "終了"}, {"r", "更新"}, {"q", "quit"}})
	}
	return body + "\n" + bottom
}

// ------------------------------------------------------------- shared bars

func (m Model) titlebar(left, right string) string {
	leftR := sTitleLeft.Render(" " + left + " ")
	rightR := sTitlebar.Render(right + " ")
	gap := m.width - ansi.StringWidth(leftR) - ansi.StringWidth(rightR)
	if gap < 1 {
		return sTitlebar.Width(m.width).Render(leftR)
	}
	return leftR + sTitlebar.Render(strings.Repeat(" ", gap)) + rightR
}

func (m Model) keybar(keys [][2]string) string {
	parts := make([]string, 0, len(keys))
	for _, kv := range keys {
		parts = append(parts, sKey.Render(kv[0])+" "+sSubtle.Render(kv[1]))
	}
	bar := " " + strings.Join(parts, sSection.Render(" · "))
	if n := m.currentNotice(); n != "" {
		notice := sAccent.Bold(true).Render(n)
		gap := m.width - ansi.StringWidth(bar) - ansi.StringWidth(notice) - 2
		if gap > 0 {
			bar += strings.Repeat(" ", gap) + notice
		}
	}
	return padANSI(bar, m.width)
}

// panel renders a rounded-border box of exactly w×h cells; the title sits on
// the first content line. Every content line is ANSI-truncated to fit and
// wrapped with the panel background so gaps between styled spans are filled.
func panel(title, subtitle string, lines []string, w, h int) string {
	innerW, innerH := w-2, h-2
	head := sPanelTitle.Render(" " + title + " ")
	if subtitle != "" {
		head += sSubtle.Render(subtitle)
	}
	content := make([]string, 0, innerH)
	content = append(content, padPanelLine(head, innerW))
	for _, l := range lines {
		if len(content) >= innerH {
			break
		}
		content = append(content, padPanelLine(l, innerW))
	}
	for len(content) < innerH {
		content = append(content, padPanelLine("", innerW))
	}
	return sPanel.Render(strings.Join(content, "\n"))
}

func sectionLine(title, subtitle string, w int) string {
	label := sSection.Render("── ") + sAccent.Render(title)
	if subtitle != "" {
		label += sSubtle.Render(" " + subtitle)
	}
	rest := w - ansi.StringWidth(label) - 1
	if rest > 0 {
		label += " " + sSection.Render(strings.Repeat("─", rest))
	}
	return label
}

// --------------------------------------------------------------- list mode

func (m Model) viewListBody() string {
	busy, waiting, unread := 0, 0, 0
	for _, s := range m.sessions {
		switch s.Status {
		case "busy":
			busy++
		case "waiting":
			waiting++
		}
		if m.seen.IsUnread(s) {
			unread++
		}
	}
	right := fmt.Sprintf("%d sessions · %d busy · %d waiting · %d unread   %s",
		len(m.sessions), busy, waiting, unread, time.Now().Format("15:04"))
	top := m.titlebar("lazyclaude", right)

	panelH := m.height - 2
	leftW := m.width / 2
	if leftW < 34 {
		leftW = 34
	}
	if leftW > 64 {
		leftW = 64
	}
	left := m.viewSessionsPanel(leftW, panelH)
	right2 := m.viewDetailPanel(m.width-leftW, panelH)
	return top + "\n" + lipgloss.JoinHorizontal(lipgloss.Top, left, right2)
}

func (m Model) viewSessionsPanel(w, h int) string {
	innerW := w - 2
	unread := 0
	for _, s := range m.sessions {
		if m.seen.IsUnread(s) {
			unread++
		}
	}
	subtitle := ""
	if unread > 0 {
		subtitle = fmt.Sprintf("· %d unread ", unread)
	}

	if len(m.sessions) == 0 {
		return panel("Sessions", subtitle, []string{
			"",
			sSubtle.Render("  実行中のセッションがありません"),
			"",
			"  " + sKey.Render("n") + sSubtle.Render(" で新規セッションを作成"),
		}, w, h)
	}

	lines := make([]string, 0, len(m.sessions))
	for i, s := range m.sessions {
		stats := m.store.ScanTranscript(s.Transcript)
		name := stats.Title
		if name == "" {
			name = format.ShortCWD(s.CWD, 40)
		}
		meta := format.Tokens(stats.Total()) + " · " + format.Age(s.UpdatedAt)
		isUnread := m.seen.IsUnread(s)
		nameW := innerW - 8 - ansi.StringWidth(meta)
		if nameW < 4 {
			nameW = 4
		}

		badge := " "
		if isUnread {
			badge = "!"
		}
		if i == m.selected {
			row := fmt.Sprintf(" ❯ %s %s %s %s", badge, statusIcon(s.Status),
				padANSI(name, nameW), meta)
			lines = append(lines, sSelected.Render(padANSI(row, innerW)))
			continue
		}
		badgeR := " "
		if isUnread {
			badgeR = sBadge.Render("!")
		}
		nameStyle := sValue
		if isUnread {
			nameStyle = sValue.Bold(true)
		} else if s.Status == "idle" {
			nameStyle = sIdle
		}
		lines = append(lines, fmt.Sprintf("   %s %s %s %s", badgeR,
			statusStyle(s.Status).Bold(true).Render(statusIcon(s.Status)),
			nameStyle.Render(padANSI(name, nameW)), sSubtle.Render(meta)))
	}
	return panel("Sessions", subtitle, lines, w, h)
}

func (m Model) viewDetailPanel(w, h int) string {
	if len(m.sessions) == 0 {
		return panel("Detail", "", []string{"", sSubtle.Render("  セッションがありません")}, w, h)
	}
	s := m.sessions[m.selected]
	innerW := w - 2
	stats := m.store.ScanTranscript(s.Transcript)
	tail := claude.TailFromFile(s.Transcript, tailProbeBytes)

	title := stats.Title
	if title == "" {
		title = "Detail"
	}

	field := func(name, value string) string {
		return "  " + sSubtle.Render(padANSI(name, 9)) + sValue.Render(value)
	}
	paneName := "tmux外"
	if m.detailPane != nil {
		paneName = m.detailPane.Name
	}
	ctx := "-"
	if tail.HasUsage {
		ctx = format.Tokens(tail.ContextTokens)
	}

	lines := []string{
		"  " + statusStyle(s.Status).Bold(true).Render(statusIcon(s.Status)+" "+s.Status) +
			sSubtle.Render(" · "+format.Age(s.UpdatedAt)+" ago"),
		"",
		field("model", format.ShortModel(tail.Model)+" · v"+s.Version),
		field("pane", fmt.Sprintf("%s · pid %d", paneName, s.PID)),
		field("dir", format.ShortCWD(s.CWD, innerW-12)),
		field("session", s.SessionID),
		"",
		sectionLine("Tokens", "", innerW),
		field("context", ctx),
		field("in / out", fmt.Sprintf("%s / %s  (%d msgs)",
			format.Tokens(stats.Input), format.Tokens(stats.Output), stats.Count)),
		field("cache", "W "+format.Tokens(stats.CacheW)+" · R "+format.Tokens(stats.CacheR)),
		field("total", format.Tokens(stats.Total())),
		"",
	}

	if m.detailPane != nil {
		lines = append(lines, sectionLine("Terminal", "· "+m.detailPane.Name+" · live", innerW))
		bodyH := (h - 2) - len(lines) - 1
		start := len(m.detailLines) - bodyH
		if start < 0 {
			start = 0
		}
		if bodyH > 0 {
			lines = append(lines, m.detailLines[start:]...)
		}
	}
	return panel(title, "", lines, w, h)
}

// -------------------------------------------------------------- viewer mode

func (m Model) viewViewerBody() string {
	title := m.store.ScanTranscript(m.viewerSession.Transcript).Title
	if title == "" {
		title = format.ShortCWD(m.viewerSession.CWD, 40)
	}
	paneName := ""
	if m.viewerPane != nil {
		paneName = m.viewerPane.Name
	}
	top := m.titlebar(paneName+" · "+title, "live 0.5s")

	bodyH := m.height - 2
	start := len(m.viewerLines) - bodyH
	if start < 0 {
		start = 0
	}
	lines := make([]string, 0, bodyH)
	for _, l := range m.viewerLines[start:] {
		lines = append(lines, ansi.Truncate(l, m.width, "")+"\x1b[m")
	}
	for len(lines) < bodyH {
		lines = append(lines, "")
	}
	return top + "\n" + strings.Join(lines, "\n")
}
