package ui

import (
	"github.com/DavidDoesDev/launchd-tui/launchd"
	"github.com/charmbracelet/lipgloss"
)

// Catppuccin Mocha palette
const (
	ctpBase    = lipgloss.Color("#1e1e2e")
	ctpMantle  = lipgloss.Color("#181825")
	ctpSurface0 = lipgloss.Color("#313244")
	ctpSurface1 = lipgloss.Color("#45475a")
	ctpOverlay0 = lipgloss.Color("#6c7086")
	ctpSubtext0 = lipgloss.Color("#a6adc8")
	ctpText     = lipgloss.Color("#cdd6f4")
	ctpGreen    = lipgloss.Color("#a6e3a1")
	ctpGreenDim = lipgloss.Color("#4a6741")
	ctpRed      = lipgloss.Color("#f38ba8")
	ctpYellow   = lipgloss.Color("#f9e2af")
	ctpBlue     = lipgloss.Color("#89b4fa")
	ctpMauve    = lipgloss.Color("#cba6f7")
)

var (
	leftPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ctpSurface1).
			Padding(0, 1)

	rightPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ctpSurface1).
			Padding(0, 1)

	barStyle = lipgloss.NewStyle().
			Background(ctpMantle).
			Foreground(ctpOverlay0).
			Padding(0, 1)

	rowStyle = lipgloss.NewStyle().
			Foreground(ctpSubtext0)

	selectedRowStyle = lipgloss.NewStyle().
				Foreground(ctpText).
				Background(ctpSurface0)

	iconRunning    = lipgloss.NewStyle().Foreground(ctpGreen)
	iconRunningDim = lipgloss.NewStyle().Foreground(ctpGreenDim)
	iconStopped    = lipgloss.NewStyle().Foreground(ctpOverlay0)
	iconErrored    = lipgloss.NewStyle().Foreground(ctpRed)
	iconUnknown    = lipgloss.NewStyle().Foreground(ctpSurface1)

	spinnerStyle = lipgloss.NewStyle().Foreground(ctpYellow)

	activeTabStyle = lipgloss.NewStyle().
			Foreground(ctpBlue).
			Bold(true).
			Underline(true)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(ctpOverlay0)

	infoLabelStyle = lipgloss.NewStyle().
			Foreground(ctpOverlay0).
			Width(12)

	infoValueStyle = lipgloss.NewStyle().
			Foreground(ctpText)

	dimStyle = lipgloss.NewStyle().
			Foreground(ctpSurface1).
			Italic(true)
)

func statusIconStyle(s launchd.Status, pulse bool) lipgloss.Style {
	switch s {
	case launchd.StatusRunning:
		if pulse {
			return iconRunningDim
		}
		return iconRunning
	case launchd.StatusStopped:
		return iconStopped
	case launchd.StatusErrored:
		return iconErrored
	default:
		return iconUnknown
	}
}
