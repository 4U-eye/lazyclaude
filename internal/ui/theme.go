package ui

import "github.com/charmbracelet/lipgloss"

// 256-color palette. lipgloss/termenv degrades gracefully on 8/16-color terminals.
var (
	cBusy   = lipgloss.Color("114") // soft green
	cWait   = lipgloss.Color("214") // orange
	cIdle   = lipgloss.Color("245") // mid gray
	cAccent = lipgloss.Color("75")  // sky blue
	cDim    = lipgloss.Color("242")
	cBorder = lipgloss.Color("238")
	cText   = lipgloss.Color("252")
	cSelBg  = lipgloss.Color("25") // deep blue selection bar
	cSelFg  = lipgloss.Color("231")
	cBadge  = lipgloss.Color("168") // pink unread badge
	cBarBg  = lipgloss.Color("236") // titlebar background
	cBarFg  = lipgloss.Color("252")
)

var (
	sTitlebar   = lipgloss.NewStyle().Background(cBarBg).Foreground(cBarFg)
	sTitleLeft  = sTitlebar.Bold(true)
	sPanel      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(cBorder)
	sPanelTitle = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	sSubtle     = lipgloss.NewStyle().Foreground(cDim)
	sValue      = lipgloss.NewStyle().Foreground(cText)
	sAccent     = lipgloss.NewStyle().Foreground(cAccent)
	sKey        = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	sSelected   = lipgloss.NewStyle().Background(cSelBg).Foreground(cSelFg).Bold(true)
	sBadge      = lipgloss.NewStyle().Background(cBadge).Foreground(cSelFg).Bold(true)
	sWarnBar    = lipgloss.NewStyle().Background(cBadge).Foreground(cSelFg).Bold(true)
	sBusy       = lipgloss.NewStyle().Foreground(cBusy)
	sWait       = lipgloss.NewStyle().Foreground(cWait)
	sIdle       = lipgloss.NewStyle().Foreground(cIdle)
	sSection    = lipgloss.NewStyle().Foreground(cBorder)
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
