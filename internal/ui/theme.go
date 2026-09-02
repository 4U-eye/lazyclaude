package ui

import "github.com/charmbracelet/lipgloss"

// Cyberpunk 256-color palette. lipgloss/termenv degrades gracefully on 8/16-color terminals.
var (
	cBusy    = lipgloss.Color("46")  // matrix neon green
	cWait    = lipgloss.Color("226") // electric yellow
	cIdle    = lipgloss.Color("103") // steel violet (readable)
	cAccent  = lipgloss.Color("51")  // neon cyan
	cDim     = lipgloss.Color("244") // readable dim
	cBorder  = lipgloss.Color("57")  // vivid purple border
	cText    = lipgloss.Color("231")
	cSelBg   = lipgloss.Color("54")  // deep magenta selection bar
	cSelFg   = lipgloss.Color("231") // white
	cBadge   = lipgloss.Color("201") // hot magenta unread badge
	cBarBg   = lipgloss.Color("233") // dark titlebar
	cBarFg   = lipgloss.Color("51")  // neon cyan
	cPanelBg = lipgloss.Color("235") // slight dark panel fill
)

var (
	sTitlebar   = lipgloss.NewStyle().Background(cBarBg).Foreground(cBarFg).Bold(true)
	sTitleLeft  = sTitlebar.Foreground(cBadge)
	sPanel      = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(cBorder).Background(cPanelBg)
	sPanelTitle = lipgloss.NewStyle().Foreground(cAccent).Background(cPanelBg).Bold(true)
	sSubtle     = lipgloss.NewStyle().Foreground(cDim).Background(cPanelBg)
	sValue      = lipgloss.NewStyle().Foreground(cText).Background(cPanelBg)
	sAccent     = lipgloss.NewStyle().Foreground(cAccent).Background(cPanelBg)
	sKey        = lipgloss.NewStyle().Foreground(cBadge).Background(cPanelBg).Bold(true)
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
		return "●"
	case "waiting":
		return "◐"
	case "idle":
		return "○"
	default:
		return "?"
	}
}
