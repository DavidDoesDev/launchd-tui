package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	width  int
	height int
}

func New() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	barHeight := 1
	panesHeight := m.height - barHeight

	// border is 2 cells (top+bottom, left+right each)
	leftWidth := m.width*30/100 - 2
	rightWidth := m.width - (leftWidth + 2) - 2
	contentHeight := panesHeight - 2

	left := leftPaneStyle.
		Width(leftWidth).
		Height(contentHeight).
		Render("Agents\n\n  (no agents configured)")

	right := rightPaneStyle.
		Width(rightWidth).
		Height(contentHeight).
		Render("[L] Logs   [I] Info\n\n  Select an agent")

	panes := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	bar := barStyle.Width(m.width).Render("↑↓ navigate · s start · x stop · r restart · tab panel · q quit")

	return lipgloss.JoinVertical(lipgloss.Left, panes, bar)
}
