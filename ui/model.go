package ui

import (
	"fmt"
	"strings"

	"github.com/DavidDoesDev/launchd-tui/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	width   int
	height  int
	agents  []config.Agent
	cursor  int
	loadErr error
}

func New() Model {
	cfg, err := config.Load()
	return Model{
		agents:  cfg.Agents,
		loadErr: err,
	}
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
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.agents)-1 {
				m.cursor++
			}
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
	leftWidth := m.width*30/100 - 2
	rightWidth := m.width - (leftWidth + 2) - 2
	contentHeight := panesHeight - 2

	left := leftPaneStyle.
		Width(leftWidth).
		Height(contentHeight).
		Render(m.renderList(leftWidth))

	right := rightPaneStyle.
		Width(rightWidth).
		Height(contentHeight).
		Render("[L] Logs   [I] Info\n\n  Select an agent")

	panes := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	bar := barStyle.Width(m.width).Render("↑↓ navigate · s start · x stop · r restart · tab panel · q quit")

	return lipgloss.JoinVertical(lipgloss.Left, panes, bar)
}

func (m Model) renderList(width int) string {
	if m.loadErr != nil {
		return fmt.Sprintf("error loading config:\n%v", m.loadErr)
	}
	if len(m.agents) == 0 {
		return "No agents configured.\n\nAdd entries to ~/.launchd-tui"
	}

	var b strings.Builder
	for i, agent := range m.agents {
		name := agent.DisplayName()
		if i == m.cursor {
			b.WriteString(selectedRowStyle.Render("  " + name))
		} else {
			b.WriteString(rowStyle.Render("  " + name))
		}
		if i < len(m.agents)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}
