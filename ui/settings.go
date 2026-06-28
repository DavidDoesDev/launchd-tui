package ui

import (
	"fmt"
	"strings"

	"github.com/DavidDoesDev/launchd-tui/config"
	tea "github.com/charmbracelet/bubbletea"
)

const numSettingsRows = 4

var pollOptions = []int{1, 2, 5, 10}

// handleSettingsKey drives the modal while it's open. Changes apply live so the
// user sees them immediately; the file is written when the modal closes.
func (m Model) handleSettingsKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", ",", "enter":
		m.showSettings = false
		_ = config.SaveSettings(m.settings)
		return m, nil
	case "up", "k":
		if m.settingsCursor > 0 {
			m.settingsCursor--
		}
	case "down", "j":
		if m.settingsCursor < numSettingsRows-1 {
			m.settingsCursor++
		}
	case "left", "h":
		return m.changeSetting(-1)
	case "right", "l", " ":
		return m.changeSetting(+1)
	}
	return m, nil
}

func (m Model) changeSetting(dir int) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.settingsCursor {
	case 0: // theme
		themes := AllThemes()
		i := wrap(indexOfTheme(themes, m.settings.Theme)+dir, len(themes))
		m.settings.Theme = themes[i].Name
		m.styles = newStyles(themes[i])
		m.spinner.Style = m.styles.spinner
	case 1: // mouse wheel
		m.settings.MouseWheel = !m.settings.MouseWheel
		if m.settings.MouseWheel {
			cmd = tea.EnableMouseCellMotion
		} else {
			cmd = tea.DisableMouse
		}
	case 2: // animations
		m.settings.Animations = !m.settings.Animations
	case 3: // poll interval
		i := indexOfInt(pollOptions, m.settings.PollIntervalSec)
		if i < 0 {
			i = 1
		}
		m.settings.PollIntervalSec = pollOptions[wrap(i+dir, len(pollOptions))]
	}
	return m, cmd
}

func (m Model) renderSettings() string {
	rows := []struct{ label, value string }{
		{"Theme", "‹ " + m.settings.Theme + " ›"},
		{"Mouse wheel", onOff(m.settings.MouseWheel)},
		{"Animations", onOff(m.settings.Animations)},
		{"Poll interval", fmt.Sprintf("‹ %ds ›", m.settings.PollIntervalSec)},
	}

	var b strings.Builder
	b.WriteString(m.styles.modalTitle.Render("Settings"))
	b.WriteString("\n\n")
	for i, r := range rows {
		cursor := "  "
		labelStyle := m.styles.modalRow
		if i == m.settingsCursor {
			cursor = m.styles.modalValue.Render("› ")
			labelStyle = m.styles.modalRowOn
		}
		label := labelStyle.Render(fmt.Sprintf("%-14s", r.label))
		b.WriteString(cursor + label + "  " + m.styles.modalValue.Render(r.value) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(m.styles.dim.Render("↑↓ move · ←→ change · esc close"))

	return m.styles.modal.Width(40).Render(b.String())
}

func onOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func wrap(i, n int) int {
	if n == 0 {
		return 0
	}
	return ((i % n) + n) % n
}

func indexOfTheme(themes []Theme, name string) int {
	for i, t := range themes {
		if t.Name == name {
			return i
		}
	}
	return 0
}

func indexOfInt(xs []int, v int) int {
	for i, x := range xs {
		if x == v {
			return i
		}
	}
	return -1
}
