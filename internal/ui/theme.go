package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Evangelion Unit-00 Kai palette (vivid blue body, white armor, red eye).
// lipgloss/termenv degrades gracefully on 8/16-color terminals.
var (
	cBusy    = lipgloss.Color("33")  // vivid blue (Eva 00 Kai body — assertive)
	cWait    = lipgloss.Color("196") // Eva red eye (attention)
	cIdle    = lipgloss.Color("60")  // muted violet-gray
	cAccent  = lipgloss.Color("45")  // bright cyan-blue HUD
	cDim     = lipgloss.Color("247") // readable dim on navy
	cBorder  = lipgloss.Color("33")  // vivid blue armor edge
	cText    = lipgloss.Color("231") // white armor plate
	cSelBg   = lipgloss.Color("19")  // navy selection (brighter than panel bg)
	cSelFg   = lipgloss.Color("195") // pale cyan-white
	cBadge   = lipgloss.Color("196") // Eva red eye (unread alert)
	cBarBg   = lipgloss.Color("232") // near-black cockpit
	cBarFg   = lipgloss.Color("45")  // bright cyan-blue label
	cPanelBg = lipgloss.Color("234") // dark chassis (neutral, less assertive)
)

// evaBorder is a half-block HUD frame (thin edges + solid quadrant corners)
// evocative of NERV tactical displays.
var evaBorder = lipgloss.Border{
	Top:         "▔",
	Bottom:      "▁",
	Left:        "▏",
	Right:       "▕",
	TopLeft:     "▛",
	TopRight:    "▜",
	BottomLeft:  "▙",
	BottomRight: "▟",
}

var (
	sTitlebar   = lipgloss.NewStyle().Background(cBarBg).Foreground(cBarFg).Bold(true)
	sTitleLeft  = sTitlebar.Foreground(cBadge)
	sPanel      = lipgloss.NewStyle().Border(evaBorder).BorderForeground(cBorder).Background(cPanelBg)
	sPanelTitle = lipgloss.NewStyle().Foreground(cAccent).Background(cPanelBg).Bold(true)
	sSubtle     = lipgloss.NewStyle().Foreground(cDim).Background(cPanelBg)
	sValue      = lipgloss.NewStyle().Foreground(cText).Background(cPanelBg)
	sAccent     = lipgloss.NewStyle().Foreground(cAccent).Background(cPanelBg)
	sKey        = lipgloss.NewStyle().Foreground(cAccent).Background(cPanelBg).Bold(true)
	sSelected   = lipgloss.NewStyle().Background(cSelBg).Foreground(cSelFg).Bold(true)
	sBadge      = lipgloss.NewStyle().Background(cBadge).Foreground(cSelFg).Bold(true)
	sWarnBar    = lipgloss.NewStyle().Background(cBadge).Foreground(cSelFg).Bold(true)
	sBusy       = lipgloss.NewStyle().Foreground(cBusy).Background(cPanelBg)
	sWait       = lipgloss.NewStyle().Foreground(cWait).Background(cPanelBg)
	sIdle       = lipgloss.NewStyle().Foreground(cIdle).Background(cPanelBg)
	sSection    = lipgloss.NewStyle().Foreground(cBorder).Background(cPanelBg)
)

func statusStyle(status string) lipgloss.Style {
	switch status {
	case "busy":
		return sBusy
	case "waiting":
		return sWait
	default:
		return sIdle
	}
}

func statusIcon(status string) string {
	switch status {
	case "busy":
		return "◆"
	case "waiting":
		return "◈"
	case "idle":
		return "◇"
	default:
		return "?"
	}
}

// statusLabel renders the raw status string as an Eva-style HUD label.
func statusLabel(status string) string {
	switch status {
	case "busy":
		return "ACTIVE"
	case "waiting":
		return "HOLD"
	case "idle":
		return "STANDBY"
	default:
		return strings.ToUpper(status)
	}
}
