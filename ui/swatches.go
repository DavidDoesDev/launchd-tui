package ui

// TEMPORARY: selection-background swatch picker. Open with the "0" key.
// Delete this file (and its hooks in model.go) once a color is chosen.

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var swatchOptions = []struct {
	hex   string
	label string
}{
	{"#1e1e2e", "base (no highlight feel)"},
	{"#28293a", "subtle lift (base→surface0 mid)"},
	{"#313244", "surface0 — Catppuccin default"},
	{"#3a3c50", "lifted surface"},
	{"#45475a", "surface1 — most prominent / grey"},
	{"#181825", "mantle — darker than base"},
	{"#2a2b3d", "neutral charcoal"},
	{"#33304a", "faint mauve tint"},
	{"#2a3340", "faint blue tint"},
	{"#2f3a37", "faint green tint"},
}

func (m Model) renderSwatches() string {
	var b strings.Builder
	b.WriteString(m.styles.modalTitle.Render("Selection background swatches"))
	b.WriteString("\n\n")
	for i, sw := range swatchOptions {
		block := lipgloss.NewStyle().Background(lipgloss.Color(sw.hex)).Render(strings.Repeat(" ", 16))
		name := lipgloss.NewStyle().Foreground(lipgloss.Color(sw.hex)).Render(sw.label)
		b.WriteString(fmt.Sprintf(" %2d  %s  %s  %s\n", i+1, block, sw.hex, name))
	}
	b.WriteString("\n")
	b.WriteString(m.styles.dim.Render("tell me the number · esc to close"))
	return m.styles.modal.Render(b.String())
}
