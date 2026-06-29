package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var helpBindings = []struct{ keys, desc string }{
	{"↑ ↓  j k", "navigate agents"},
	{"s x r", "start · stop · restart"},
	{"tab", "switch Logs / Info"},
	{"pgup pgdn", "scroll logs"},
	{"G", "jump to latest (resume tail)"},
	{"wheel", "scroll logs (when mouse on)"},
	{",", "settings"},
	{"?", "this help"},
	{"q", "quit"},
}

func (m Model) renderHelp() string {
	keyStyle := lipgloss.NewStyle().Foreground(m.styles.theme.Blue).Bold(true).Width(12)

	var b strings.Builder
	b.WriteString(m.styles.modalTitle.Render("Keys"))
	b.WriteString("\n\n")
	for _, k := range helpBindings {
		b.WriteString(keyStyle.Render(k.keys) + "  " + m.styles.modalRow.Render(k.desc) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(m.styles.dim.Render("esc to close"))
	return m.styles.modal.Render(b.String())
}
