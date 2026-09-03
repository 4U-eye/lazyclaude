// Package ui implements the interactive TUI (bubbletea).
package ui

import (
	"os"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/4U-eye/lazyclaude/internal/claude"
	"github.com/4U-eye/lazyclaude/internal/config"
	"github.com/4U-eye/lazyclaude/internal/tmux"
)

type mode int

const (
	modeList mode = iota
	modeViewer
	modeInput
	modeConfirm
)

type inputPurpose int

const (
	inputNewDir inputPurpose = iota
	inputInstruction
)

const (
	tickInterval   = 300 * time.Millisecond
	noticeDuration = 4 * time.Second
	paneCacheOK    = 10 * time.Second
	paneCacheMiss  = 5 * time.Second
	tailProbeBytes = 64 * 1024
	minWidth       = 50
	minHeight      = 8
)

type paneEntry struct {
	pane    *tmux.Pane
	expires time.Time
}

// Model is the whole application state (single bubbletea model, mode-switched).
type Model struct {
	store    *claude.Store
	seenPath string
	seen     claude.Seen

	sessions []claude.Session
	selected int
	width    int
	height   int

	mode mode

	// viewer state
	viewerSession claude.Session
	viewerPane    *tmux.Pane
	viewerLines   []string

	// detail pane live preview
	detailPane  *tmux.Pane
	detailLines []string

	input        textinput.Model
	inputFor     inputPurpose
	confirmMsg   string
	confirmParam claude.Session

	notice      string
	noticeUntil time.Time

	paneCache map[int]paneEntry

	tickPhase   int       // increments each tick; used for blinking accents
	startedAt   time.Time // process start time, used for runtime clock in titlebar
	splashUntil time.Time // NERV startup splash is shown until this instant

	cfg config.Config
}

// New builds the initial model.
func New() Model {
	ti := textinput.New()
	ti.Prompt = "▶ "
	ti.CharLimit = 2000
	return Model{
		store:       claude.DefaultStore(),
		seenPath:    claude.DefaultSeenPath(),
		seen:        claude.LoadSeen(claude.DefaultSeenPath()),
		input:       ti,
		paneCache:   map[int]paneEntry{},
		startedAt:   time.Now(),
		splashUntil: time.Now().Add(2500 * time.Millisecond),
		cfg:         config.Load(),
	}
}

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// attachDoneMsg fires when an external `tmux attach` (jump from outside tmux) ends.
type attachDoneMsg struct{}

func (m Model) Init() tea.Cmd {
	return tick()
}

func (m *Model) refresh() {
	m.sessions = m.store.Load()
	m.clampSelected()
	// 新規セッションは既読で初期化（初回起動時の全件未読ノイズを防ぐ）
	for _, s := range m.sessions {
		if _, ok := m.seen[s.SessionID]; !ok && s.SessionID != "" {
			m.seen.MarkSeen(s)
		}
	}
}

func (m *Model) clampSelected() {
	if m.selected >= len(m.sessions) {
		m.selected = len(m.sessions) - 1
	}
	if m.selected < 0 {
		m.selected = 0
	}
}

// cachedPane resolves the tmux pane for a pid, throttling ps/tmux calls.
func (m *Model) cachedPane(pid int) *tmux.Pane {
	if e, ok := m.paneCache[pid]; ok && time.Now().Before(e.expires) {
		return e.pane
	}
	p := tmux.FindPane(pid)
	ttl := paneCacheOK
	if p == nil {
		ttl = paneCacheMiss
	}
	m.paneCache[pid] = paneEntry{pane: p, expires: time.Now().Add(ttl)}
	return p
}

func (m *Model) setNotice(text string) {
	m.notice = text
	m.noticeUntil = time.Now().Add(noticeDuration)
}

func (m *Model) currentNotice() string {
	if time.Now().Before(m.noticeUntil) {
		return m.notice
	}
	return ""
}

func (m *Model) capture() {
	switch m.mode {
	case modeViewer:
		if m.viewerPane != nil {
			if lines, err := tmux.Capture(m.viewerPane.ID, true); err == nil {
				m.viewerLines = lines
			}
		}
	default:
		m.detailPane, m.detailLines = nil, nil
		if len(m.sessions) > 0 {
			if p := m.cachedPane(m.sessions[m.selected].PID); p != nil {
				m.detailPane = p
				if lines, err := tmux.Capture(p.ID, true); err == nil {
					m.detailLines = lines
				}
			}
		}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		m.tickPhase++
		m.refresh()
		m.capture()
		return m, tick()

	case attachDoneMsg:
		// attachから戻った = セッションを確認した扱いでビューアを閉じる
		m.closeViewerMarkingSeen()
		return m, nil

	case tea.KeyMsg:
		// splash中はどのキーでもスキップして main UI へ
		if time.Now().Before(m.splashUntil) {
			m.splashUntil = time.Now()
			return m, nil
		}
		switch m.mode {
		case modeInput:
			return m.updateInput(msg)
		case modeConfirm:
			return m.updateConfirm(msg)
		case modeViewer:
			return m.updateViewer(msg)
		default:
			return m.updateList(msg)
		}
	}
	return m, nil
}

func (m *Model) closeViewerMarkingSeen() {
	m.seen.MarkSeen(m.viewerSession)
	m.seen.Save(m.seenPath) //nolint:errcheck // 保存失敗しても監視は継続
	m.mode = modeList
	m.viewerPane = nil
	m.viewerLines = nil
}

func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "ctrl+c":
		m.seen.Save(m.seenPath) //nolint:errcheck
		return m, tea.Quit
	case "j", "down":
		m.selected++
		m.clampSelected()
	case "k", "up":
		m.selected--
		m.clampSelected()
	case "g":
		m.selected = 0
	case "G":
		m.selected = len(m.sessions) - 1
		m.clampSelected()
	case "r":
		m.refresh()
		m.capture()
	case "enter":
		if len(m.sessions) == 0 {
			break
		}
		s := m.sessions[m.selected]
		p := m.cachedPane(s.PID)
		if p == nil {
			m.setNotice("✗ UNIT OFF-GRID")
			break
		}
		m.mode = modeViewer
		m.viewerSession = s
		m.viewerPane = p
		m.capture()
	case "n":
		m.mode = modeInput
		m.inputFor = inputNewDir
		m.input.Placeholder = "CWD (blank=" + m.cfg.NewDir + ")"
		m.input.SetValue("")
		m.input.Focus()
	case "x":
		if len(m.sessions) == 0 {
			break
		}
		s := m.sessions[m.selected]
		name := m.store.ScanTranscript(s.Transcript).Title
		if name == "" {
			name = s.CWD
		}
		m.mode = modeConfirm
		m.confirmMsg = "TERMINATE UNIT? — " + name
		m.confirmParam = s
	}
	return m, nil
}

func (m Model) updateViewer(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.closeViewerMarkingSeen()
	case "i":
		m.mode = modeInput
		m.inputFor = inputInstruction
		m.input.Placeholder = "TRANSMIT MESSAGE · ENTER"
		m.input.SetValue("")
		m.input.Focus()
	case "a":
		pane := m.viewerPane
		tmux.SelectPane(pane.ID)
		if tmux.InsideTmux() {
			tmux.SwitchClient(pane.ID) //nolint:errcheck
			m.closeViewerMarkingSeen()
			return m, nil
		}
		return m, tea.ExecProcess(tmux.AttachCmd(pane.ID), func(error) tea.Msg {
			return attachDoneMsg{}
		})
	}
	return m, nil
}

func (m Model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.input.Blur()
		if m.inputFor == inputInstruction {
			m.mode = modeViewer
		} else {
			m.mode = modeList
		}
		return m, nil
	case "enter":
		value := m.input.Value()
		m.input.Blur()
		switch m.inputFor {
		case inputNewDir:
			m.mode = modeList
			dir := value
			if dir == "" {
				dir = m.cfg.NewDir
			}
			paneID, err := tmux.CreateSession(m.cfg.ClaudeSession, dir, m.cfg.ClaudeCommand)
			if err != nil {
				m.setNotice("✗ " + err.Error())
				return m, nil
			}
			m.setNotice("✓ UNIT DEPLOYED")
			tmux.SelectPane(paneID)
			if tmux.InsideTmux() {
				tmux.SwitchClient(paneID) //nolint:errcheck
				return m, nil
			}
			return m, tea.ExecProcess(tmux.AttachCmd(paneID), func(error) tea.Msg {
				return attachDoneMsg{}
			})
		case inputInstruction:
			m.mode = modeViewer
			if value == "" {
				return m, nil
			}
			if err := tmux.Send(m.viewerPane.ID, value); err != nil {
				m.setNotice("✗ TRANSMISSION FAILED")
			} else {
				m.setNotice("✓ TRANSMITTED")
			}
			m.capture()
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if s := msg.String(); s == "y" || s == "Y" {
		ok := false
		if p := m.cachedPane(m.confirmParam.PID); p != nil {
			ok = tmux.KillPane(p.ID) == nil
		} else if proc, err := os.FindProcess(m.confirmParam.PID); err == nil {
			ok = proc.Signal(os.Interrupt) == nil
		}
		if ok {
			m.setNotice("✓ UNIT TERMINATED")
		} else {
			m.setNotice("✗ TERMINATION FAILED")
		}
		delete(m.paneCache, m.confirmParam.PID)
		time.Sleep(300 * time.Millisecond) // プロセス終了とレジストリ反映を待つ
		m.refresh()
	}
	m.mode = modeList
	return m, nil
}
