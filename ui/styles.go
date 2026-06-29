package ui

import (
	"github.com/DavidDoesDev/launchd-tui/launchd"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

// blend mixes two hex colors by t (0 = a, 1 = b).
func blend(a, b lipgloss.Color, t float64) lipgloss.Color {
	ca, err1 := colorful.Hex(string(a))
	cb, err2 := colorful.Hex(string(b))
	if err1 != nil || err2 != nil {
		return a
	}
	return lipgloss.Color(ca.BlendRgb(cb, t).Hex())
}

// lighten mixes a color toward white by amt.
func lighten(c lipgloss.Color, amt float64) lipgloss.Color {
	return blend(c, lipgloss.Color("#ffffff"), amt)
}

// Styles holds every lipgloss.Style the UI renders with, derived from a Theme.
// It's rebuilt (newStyles) whenever the active theme changes, so rendering code
// reads m.styles.X instead of package-level globals — themes become swappable
// at runtime.
type Styles struct {
	theme   Theme
	selText lipgloss.Color // whiter text for the selected card name

	leftPane  lipgloss.Style
	rightPane lipgloss.Style
	bar       lipgloss.Style
	row       lipgloss.Style

	iconRunning lipgloss.Style
	iconStopped lipgloss.Style
	iconErrored lipgloss.Style
	iconUnknown lipgloss.Style

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
		theme:   t,
		selText: lighten(t.Text, 0.45),

		leftPane: lipgloss.NewStyle().Padding(0, 1),
		rightPane: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).BorderForeground(t.Surface1).
			Padding(1, 2), // ~2× the usual pane padding
		bar: lipgloss.NewStyle().
			Background(t.Mantle).Foreground(t.Overlay0).Padding(0, 1),
		row: lipgloss.NewStyle().Foreground(t.Subtext0),

		iconRunning: lipgloss.NewStyle().Foreground(t.Green),
		iconStopped: lipgloss.NewStyle().Foreground(t.Overlay0),
		iconErrored: lipgloss.NewStyle().Foreground(t.Red),
		iconUnknown: lipgloss.NewStyle().Foreground(t.Surface1),

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
		logDividerLabel: lipgloss.NewStyle().Foreground(t.Blue).Bold(true),

		statusRunning: lipgloss.NewStyle().Foreground(t.Green),
		statusStopped: lipgloss.NewStyle().Foreground(t.Subtext0),
		statusErrored: lipgloss.NewStyle().Foreground(t.Red),
		statusUnknown: lipgloss.NewStyle().Foreground(t.Overlay0),

		modal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).BorderForeground(t.Mauve).
			Padding(1, 2), // transparent interior to match the (borderless) app bg
		modalTitle: lipgloss.NewStyle().Foreground(t.Mauve).Bold(true),
		modalRow:   lipgloss.NewStyle().Foreground(t.Subtext0),
		modalRowOn: lipgloss.NewStyle().Foreground(t.Text).Bold(true),
		modalValue: lipgloss.NewStyle().Foreground(t.Blue),
	}
}

func (s Styles) statusIcon(status launchd.Status) lipgloss.Style {
	switch status {
	case launchd.StatusRunning:
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
