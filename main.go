package main

import (
	"fmt"
	"os"

	"github.com/DavidDoesDev/launchd-tui/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func main() {
	// Force 24-bit color so subtle tones (e.g. surface0 #313244) render exactly
	// even when the terminal doesn't advertise COLORTERM=truecolor — otherwise
	// lipgloss downsamples to the 256-color palette and dark navies read as grey.
	lipgloss.SetColorProfile(termenv.TrueColor)

	// Mouse reporting is enabled/disabled at runtime from Init and the settings
	// menu (see Model), so it isn't forced on here.
	p := tea.NewProgram(ui.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
