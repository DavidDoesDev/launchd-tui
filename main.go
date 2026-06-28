package main

import (
	"fmt"
	"os"

	"github.com/DavidDoesDev/launchd-tui/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Mouse reporting is enabled/disabled at runtime from Init and the settings
	// menu (see Model), so it isn't forced on here.
	p := tea.NewProgram(ui.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
