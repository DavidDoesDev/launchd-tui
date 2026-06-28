package ui

import (
	"github.com/DavidDoesDev/launchd-tui/launchd"
	"github.com/charmbracelet/lipgloss"
)

// Styles holds every lipgloss.Style the UI renders with, derived from a Theme.
// It's rebuilt (newStyles) whenever the active theme changes, so rendering code
// reads m.styles.X instead of package-level globals — themes become swappable
// at runtime.
type Styles struct {
	theme Theme

	leftPane    lipgloss.Style
	rightPane   lipgloss.Style
	bar         lipgloss.Style
	row         lipgloss.Style
	selectedRow lipgloss.Style

	iconRunning    lipgloss.Style
	iconRunningDim lipgloss.Style
	iconStopped    lipgloss.Style
	iconErrored    lipgloss.Style
	iconUnknown    lipgloss.Style

	spinner     lipgloss.Style
	activeTab   lipgloss.Style
	inactiveTab lipgloss.Style
	infoLabel   lipgloss.Style
	infoValue   lipgloss.Style
	dim         lipgloss.Style

	logTimestamp lipgloss.Style
	logError     lipgloss.Style
	logWarn      lipgloss.Style
	logSuccess   lipgloss.Style
	logDefault   lipgloss.Style

	logDividerRule  lipgloss.Style
	logDividerLabel lipgloss.Style

	statusRunning lipgloss.Style
	statusStopped lipgloss.Style
	statusErrored lipgloss.Style
	statusUnknown lipgloss.Style

	modal      lipgloss.Style
	modalTitle lipgloss.Style
	modalRow   lipgloss.Style
	modalRowOn lipgloss.Style
	modalValue lipgloss.Style
}

func newStyles(t Theme) Styles {
	return Styles{
		theme: t,

		leftPane: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).BorderForeground(t.Surface1).Padding(0, 1),
		rightPane: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).BorderForeground(t.Surface1).Padding(0, 1),
		bar: lipgloss.NewStyle().
			Background(t.Mantle).Foreground(t.Overlay0).Padding(0, 1),
		row:         lipgloss.NewStyle().Foreground(t.Subtext0),
		selectedRow: lipgloss.NewStyle().Foreground(t.Text).Background(t.Surface0),

		iconRunning:    lipgloss.NewStyle().Foreground(t.Green),
		iconRunningDim: lipgloss.NewStyle().Foreground(t.GreenDim),
		iconStopped:    lipgloss.NewStyle().Foreground(t.Overlay0),
		iconErrored:    lipgloss.NewStyle().Foreground(t.Red),
		iconUnknown:    lipgloss.NewStyle().Foreground(t.Surface1),

		spinner:     lipgloss.NewStyle().Foreground(t.Yellow),
		activeTab:   lipgloss.NewStyle().Foreground(t.Blue).Bold(true).Underline(true),
		inactiveTab: lipgloss.NewStyle().Foreground(t.Overlay0),
		infoLabel:   lipgloss.NewStyle().Foreground(t.Overlay0).Width(12),
		infoValue:   lipgloss.NewStyle().Foreground(t.Text),
		dim:         lipgloss.NewStyle().Foreground(t.Surface1).Italic(true),

		logTimestamp: lipgloss.NewStyle().Foreground(t.Overlay0),
		logError:     lipgloss.NewStyle().Foreground(t.Red),
		logWarn:      lipgloss.NewStyle().Foreground(t.Yellow),
		logSuccess:   lipgloss.NewStyle().Foreground(t.Green),
		logDefault:   lipgloss.NewStyle().Foreground(t.Subtext0),

		logDividerRule:  lipgloss.NewStyle().Foreground(t.Surface1),
		logDividerLabel: lipgloss.NewStyle().Foreground(t.Mauve).Bold(true),

		statusRunning: lipgloss.NewStyle().Foreground(t.Green),
		statusStopped: lipgloss.NewStyle().Foreground(t.Subtext0),
		statusErrored: lipgloss.NewStyle().Foreground(t.Red),
		statusUnknown: lipgloss.NewStyle().Foreground(t.Overlay0),

		modal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).BorderForeground(t.Mauve).
			Background(t.Base).Padding(1, 2),
		modalTitle: lipgloss.NewStyle().Foreground(t.Mauve).Bold(true),
		modalRow:   lipgloss.NewStyle().Foreground(t.Subtext0),
		modalRowOn: lipgloss.NewStyle().Foreground(t.Text).Bold(true),
		modalValue: lipgloss.NewStyle().Foreground(t.Blue),
	}
}

func (s Styles) statusIcon(status launchd.Status, pulse bool) lipgloss.Style {
	switch status {
	case launchd.StatusRunning:
		if pulse {
			return s.iconRunningDim
		}
		return s.iconRunning
	case launchd.StatusStopped:
		return s.iconStopped
	case launchd.StatusErrored:
		return s.iconErrored
	default:
		return s.iconUnknown
	}
}

func (s Styles) statusLabel(status launchd.Status) lipgloss.Style {
	switch status {
	case launchd.StatusRunning:
		return s.statusRunning
	case launchd.StatusStopped:
		return s.statusStopped
	case launchd.StatusErrored:
		return s.statusErrored
	default:
		return s.statusUnknown
	}
}

// bgIf returns the style with a background applied when on (used to fill a
// selected card's full width without the SGR-reset holes that a single outer
// Render would leave over inner-colored segments).
func bgIf(st lipgloss.Style, on bool, c lipgloss.Color) lipgloss.Style {
	if on {
		return st.Background(c)
	}
	return st
}
