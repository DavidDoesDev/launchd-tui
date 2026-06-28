package ui

import (
	"github.com/DavidDoesDev/launchd-tui/launchd"
	"github.com/charmbracelet/lipgloss"
)

var (
	leftPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	rightPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	barStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("245")).
			Padding(0, 1)

	rowStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250"))

	selectedRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("255")).
				Background(lipgloss.Color("237")).
				Bold(true)

	iconRunning  = lipgloss.NewStyle().Foreground(lipgloss.Color("114")) // green
	iconStopped  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // muted
	iconErrored  = lipgloss.NewStyle().Foreground(lipgloss.Color("203")) // red
	iconUnknown  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // muted
)

func statusIconStyle(s launchd.Status) lipgloss.Style {
	switch s {
	case launchd.StatusRunning:
		return iconRunning
	case launchd.StatusStopped:
		return iconStopped
	case launchd.StatusErrored:
		return iconErrored
	default:
		return iconUnknown
	}
}
