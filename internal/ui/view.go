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
const panelBgSGR = "\x1b[48;5;234m"

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
		return sWait.Render("VIEWPORT TOO SMALL")
	}
	if time.Now().Before(m.splashUntil) {
		return m.viewSplash()
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
		bottom = sWarnBar.Width(m.width).Render(" ▚▞ ⚠ " + m.confirmMsg + "  [y/N] ⚠ ▞▚")
	case modeViewer:
		bottom = m.keybar([][2]string{{"i", "TRANSMIT"}, {"a", "JUMP"}, {"q", "BACK"}})
	default:
		bottom = m.keybar([][2]string{
			{"j/k", "SCAN"}, {"enter", "LINK"}, {"n", "SPAWN"},
			{"x", "ABORT"}, {"r", "SYNC"}, {"q", "EXIT"}})
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
	return renderPanel(sPanel, title, subtitle, lines, w, h)
}

// renderPanel is panel() with a caller-supplied outer style.
func renderPanel(style lipgloss.Style, title, subtitle string, lines []string, w, h int) string {
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
	return style.Render(strings.Join(content, "\n"))
}

func sectionLine(title, subtitle string, w int) string {
	label := sAccent.Bold(true).Render("▍ [ "+title+" ]")
	if subtitle != "" {
		label += sSubtle.Render(" " + subtitle)
	}
	label += " "
	rest := w - ansi.StringWidth(label) - 1
	if rest > 0 {
		label += sSection.Render(strings.Repeat("━", rest))
	}
	return label
}

// syncMeter renders an Eva-style sync-ratio gauge for the current context.
// 200k tokens is treated as full sync; values above show a red over-sync warning.
func syncMeter(context int64, w int) string {
	const cap = 200_000
	if w <= 0 {
		w = 10
	}
	ratio := float64(context) / float64(cap)
	filled := int(ratio * float64(w))
	if filled > w {
		filled = w
	}
	if filled < 0 {
		filled = 0
	}
	fill := sBusy
	if ratio > 1.0 {
		fill = sWait
	}
	bar := fill.Render(strings.Repeat("█", filled)) + sSubtle.Render(strings.Repeat("░", w-filled))
	pct := int(ratio * 100)
	return bar + " " + sValue.Render(fmt.Sprintf("%d%%", pct))
}

// ratioMeter renders a fixed-width gauge (0.0-1.0), used for cache-hit etc.
func ratioMeter(ratio float64, w int) string {
	if w <= 0 {
		w = 10
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(w))
	if filled > w {
		filled = w
	}
	bar := sAccent.Render(strings.Repeat("█", filled)) + sSubtle.Render(strings.Repeat("░", w-filled))
	return bar + " " + sValue.Render(fmt.Sprintf("%d%%", int(ratio*100)))
}

// sparkline renders a compact bar chart from recent samples. Empty samples
// render as a flat baseline so the layout doesn't shift.
func sparkline(vals []int64, w int) string {
	bars := []rune("▁▂▃▄▅▆▇█")
	if w <= 0 {
		w = 10
	}
	if len(vals) == 0 {
		return sSubtle.Render(strings.Repeat("▁", w))
	}
	var maxv int64
	for _, v := range vals {
		if v > maxv {
			maxv = v
		}
	}
	if maxv == 0 {
		return sSubtle.Render(strings.Repeat("▁", w))
	}
	n := len(vals)
	start := 0
	if n > w {
		start = n - w
	}
	slice := vals[start:]
	var b strings.Builder
	for i := 0; i < w-len(slice); i++ {
		b.WriteRune(' ')
	}
	for _, v := range slice {
		idx := int(float64(v) / float64(maxv) * float64(len(bars)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(bars) {
			idx = len(bars) - 1
		}
		b.WriteRune(bars[idx])
	}
	return sBusy.Render(b.String())
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
	uptime := time.Since(m.startedAt)
	uptimeStr := fmt.Sprintf("T+%02d:%02d:%02d",
		int(uptime.Hours()), int(uptime.Minutes())%60, int(uptime.Seconds())%60)
	right := fmt.Sprintf("SYNC %d ▍ ACTIVE %d ▍ HOLD %d ▍ ALERT %d ▍ %s ▍ %s",
		len(m.sessions), busy, waiting, unread, uptimeStr, time.Now().Format("20060102-1504"))
	top := m.titlebar("◤◢ NERV.LAZYCLAUDE", right)

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
		subtitle = fmt.Sprintf("· %d ALERT ", unread)
	}

	if len(m.sessions) == 0 {
		return renderPanel(sPanel, "UNITS", subtitle, []string{
			"",
			sSubtle.Render("  NO UNITS DEPLOYED"),
			"",
			"  " + sKey.Render("n") + sSubtle.Render(" to launch new unit"),
		}, w, h)
	}

	lines := make([]string, 0, len(m.sessions))
	for i, s := range m.sessions {
		stats := m.store.ScanTranscript(s.Transcript)
		name := stats.Title
		if name == "" {
			name = format.ShortCWD(s.CWD, 40)
		}
		code := fmt.Sprintf("U-%02d", i+1)
		meta := format.Tokens(stats.Total()) + " · " + format.Age(s.UpdatedAt)
		isUnread := m.seen.IsUnread(s)
		nameW := innerW - 14 - ansi.StringWidth(meta)
		if nameW < 4 {
			nameW = 4
		}

		pulseOn := m.tickPhase%2 == 0
		badge := "  "
		if isUnread {
			// 色シフト型パルス: 赤 ↔ 黄 で500ms周期。常に表示されるので視認性が高い。
			style := sBadge
			if pulseOn {
				style = lipgloss.NewStyle().Background(cWait).Foreground(cText).Bold(true)
			}
			badge = style.Render(" ! ")
		}
		// 左端インジケータ: 選択=明シアン、未読=赤（両方steady）。位置と警戒を強調。
		leftBar := "  "
		if i == m.selected {
			leftBar = sAccent.Bold(true).Render("▐▶")
		} else if isUnread {
			leftBar = sBadge.Bold(true).Render("▐ ")
		}
		// busy アイコンは phase で青→シアンに揺らぐパルス
		icon := statusIcon(s.Status)
		iconStyle := statusStyle(s.Status).Bold(true)
		if s.Status == "busy" && pulseOn {
			iconStyle = sAccent.Bold(true)
		}

		if i == m.selected {
			// 選択行: sSelected bg で全体を囲む（左バーは外側で色を保持）
			innerRow := fmt.Sprintf(" %s %s %s %s %s", badge, code, icon,
				padANSI(name, nameW), meta)
			lines = append(lines, leftBar+sSelected.Render(padANSI(innerRow, innerW-2)))
			continue
		}
		nameStyle := sValue
		if isUnread {
			nameStyle = sValue.Bold(true)
		} else if s.Status == "idle" {
			nameStyle = sIdle
		}
		lines = append(lines, fmt.Sprintf("%s %s %s %s %s %s", leftBar, badge,
			sAccent.Render(code),
			iconStyle.Render(icon),
			nameStyle.Render(padANSI(name, nameW)), sSubtle.Render(meta)))
	}
	return renderPanel(sPanel, "UNITS", subtitle, lines, w, h)
}

func (m Model) viewDetailPanel(w, h int) string {
	if len(m.sessions) == 0 {
		return renderPanel(sPanel, "DIAG.CORE", "", []string{"", sSubtle.Render("  NO UNIT SELECTED")}, w, h)
	}
	s := m.sessions[m.selected]
	innerW := w - 2
	stats := m.store.ScanTranscript(s.Transcript)
	tail := claude.TailFromFile(s.Transcript, tailProbeBytes)

	title := stats.Title
	if title == "" {
		title = "DIAG.CORE"
	}

	field := func(name, value string) string {
		return "  " + sSubtle.Render(padANSI(name, 9)) + sValue.Render(value)
	}
	paneName := "OFF-GRID"
	if m.detailPane != nil {
		paneName = m.detailPane.Name
	}
	ctx := "-"
	if tail.HasUsage {
		ctx = format.Tokens(tail.ContextTokens)
	}

	statPill := sSelected.Render(" " + statusIcon(s.Status) + " " + statusLabel(s.Status) + " ")
	agePill := sSelected.Render(" " + format.Age(s.UpdatedAt) + " ago ")
	syncVal := ctx
	if tail.HasUsage {
		syncVal = syncMeter(tail.ContextTokens, 10) + "  " + sValue.Render(ctx)
	}
	hitVal := "-"
	totalIn := stats.Input + stats.CacheR + stats.CacheW
	if totalIn > 0 {
		hitVal = ratioMeter(float64(stats.CacheR)/float64(totalIn), 10)
	}
	trendVal := sparkline(tail.ContextHistory, 20)

	lines := []string{
		"  " + statPill + "  " + agePill,
		"",
		field("MDL", format.ShortModel(tail.Model)+" · v"+s.Version),
		field("TTY", fmt.Sprintf("%s · pid %d", paneName, s.PID)),
		field("CWD", format.ShortCWD(s.CWD, innerW-12)),
		field("SID", s.SessionID),
		"",
		sectionLine("SYNC.CORE", "", innerW),
		field("SYNC", syncVal),
		field("HIT", hitVal),
		field("TREND", trendVal),
		field("I/O", fmt.Sprintf("%s / %s  (%d msgs)",
			format.Tokens(stats.Input), format.Tokens(stats.Output), stats.Count)),
		field("CACHE", "W "+format.Tokens(stats.CacheW)+" · R "+format.Tokens(stats.CacheR)),
		field("TOTAL", format.Tokens(stats.Total())),
		"",
	}

	if m.detailPane != nil {
		lines = append(lines, sectionLine("LINK.LIVE", "· "+m.detailPane.Name, innerW))
		bodyH := (h - 2) - len(lines) - 1
		start := len(m.detailLines) - bodyH
		if start < 0 {
			start = 0
		}
		if bodyH > 0 {
			lines = append(lines, m.detailLines[start:]...)
		}
	}
	return renderPanel(sPanel, title, "", lines, w, h)
}

// --------------------------------------------------------------- splash mode

// nervLogo — block-ASCII "NERV" wordmark. 6 rows tall, ~46 cols wide.
var nervLogo = []string{
	" ███╗   ██╗ ███████╗ ██████╗  ██╗   ██╗",
	" ████╗  ██║ ██╔════╝ ██╔══██╗ ██║   ██║",
	" ██╔██╗ ██║ █████╗   ██████╔╝ ██║   ██║",
	" ██║╚██╗██║ ██╔══╝   ██╔══██╗ ╚██╗ ██╔╝",
	" ██║ ╚████║ ███████╗ ██║  ██║  ╚████╔╝ ",
	" ╚═╝  ╚═══╝ ╚══════╝ ╚═╝  ╚═╝   ╚═══╝  ",
}

func (m Model) viewSplash() string {
	// Logo colour pulses red ↔ yellow like an Eva warning beacon.
	logoColor := cBadge
	if m.tickPhase%2 == 0 {
		logoColor = cWait
	}
	logoStyle := lipgloss.NewStyle().Foreground(logoColor).Bold(true)
	sub1Style := sAccent.Bold(true)
	sub2Style := sSubtle

	// Pulse the status text so it clearly reads as "boot animation".
	statuses := []string{
		"SYSTEM ONLINE · MAGI READY",
		"SYNC CONFIRMED · MAGI READY",
		"AT FIELD DEPLOYED · MAGI READY",
		"SYNC CONFIRMED · MAGI READY",
	}
	status := statuses[m.tickPhase%len(statuses)]

	logoBlock := make([]string, 0, len(nervLogo))
	for _, l := range nervLogo {
		logoBlock = append(logoBlock, lipgloss.PlaceHorizontal(m.width, lipgloss.Center, logoStyle.Render(l)))
	}
	subtitle := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, sub1Style.Render("L A Z Y C L A U D E"))
	statusLn := lipgloss.PlaceHorizontal(m.width, lipgloss.Center, sub2Style.Render("▍ "+status+" ▍"))

	block := append([]string{}, logoBlock...)
	block = append(block, "", subtitle, "", statusLn)

	padTop := (m.height - len(block)) / 2
	if padTop < 0 {
		padTop = 0
	}
	out := make([]string, 0, m.height)
	for i := 0; i < padTop; i++ {
		out = append(out, "")
	}
	out = append(out, block...)
	for len(out) < m.height {
		out = append(out, "")
	}
	return strings.Join(out, "\n")
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
	top := m.titlebar(paneName+" · "+title, "LINK LIVE · 0.5s")

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
