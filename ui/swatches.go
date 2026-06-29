package ui

// TEMPORARY: color swatch reference. Open with the "0" key.
// Delete this file (and its hooks in model.go) once it's no longer needed.

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type swatch struct {
	col   lipgloss.Color
	label string
}

// buildSwatches returns the full Catppuccin Mocha palette (the theme we use),
// dark→light, so every shade is visible. herdr's selected-tab navy is surface0
// (#313244); its content-pane border is base (#1e1e2e).
func buildSwatches(Theme) []swatch {
	return []swatch{
		{"#11111b", "crust"},
		{"#181825", "mantle"},
		{"#1e1e2e", "base  ← herdr pane border"},
		{"#313244", "surface0  ← herdr selected tab"},
		{"#45475a", "surface1"},
		{"#585b70", "surface2"},
		{"#6c7086", "overlay0"},
		{"#7f849c", "overlay1"},
		{"#9399b2", "overlay2"},
		{"#a6adc8", "subtext0"},
		{"#bac2de", "subtext1"},
		{"#cdd6f4", "text"},
		{"#f5e0dc", "rosewater"},
		{"#f2cdcd", "flamingo"},
		{"#f5c2e7", "pink"},
		{"#cba6f7", "mauve"},
		{"#f38ba8", "red"},
		{"#eba0ac", "maroon"},
		{"#fab387", "peach"},
		{"#f9e2af", "yellow"},
		{"#a6e3a1", "green"},
		{"#94e2d5", "teal"},
		{"#89dceb", "sky"},
		{"#74c7ec", "sapphire"},
		{"#89b4fa", "blue"},
		{"#b4befe", "lavender"},
	}
}

func (m Model) renderSwatches() string {
	var b strings.Builder
	b.WriteString(m.styles.modalTitle.Render("Swatches"))
	b.WriteString("\n\n")
	for i, sw := range buildSwatches(m.styles.theme) {
		block := lipgloss.NewStyle().Background(sw.col).Render(strings.Repeat(" ", 10))
		b.WriteString(fmt.Sprintf(" %2d  %s  %-8s  %s\n",
			i+1, block, string(sw.col), m.styles.dim.Render(sw.label)))
	}
	b.WriteString("\n")
	b.WriteString(m.styles.dim.Render("tell me the number · esc to close"))
	return m.styles.modal.Render(b.String())
}
