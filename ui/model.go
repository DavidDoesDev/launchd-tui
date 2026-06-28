package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/DavidDoesDev/launchd-tui/config"
	"github.com/DavidDoesDev/launchd-tui/launchd"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const pollInterval = 2 * time.Second

type pollMsg struct{}

type Model struct {
	width   int
	height  int
	agents  []config.Agent
	states  []launchd.AgentState
	cursor  int
	loadErr error
}

func New() Model {
	cfg, err := config.Load()
	m := Model{
		agents:  cfg.Agents,
		states:  make([]launchd.AgentState, len(cfg.Agents)),
		loadErr: err,
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchAllStates(m.agents), pollCmd())
}

func pollCmd() tea.Cmd {
	return tea.Tick(pollInterval, func(time.Time) tea.Msg {
		return pollMsg{}
	})
}

func fetchAllStates(agents []config.Agent) tea.Cmd {
	return func() tea.Msg {
		states := make([]launchd.AgentState, len(agents))
		for i, a := range agents {
			states[i] = launchd.GetState(a.Label)
		}
		return states
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case []launchd.AgentState:
		m.states = msg

	case pollMsg:
		return m, tea.Batch(fetchAllStates(m.agents), pollCmd())

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
		Render(m.renderDetail())

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
		var state launchd.AgentState
		if i < len(m.states) {
			state = m.states[i]
		}

		icon := launchd.StatusIcon(state.Status)
		iconStyled := statusIconStyle(state.Status).Render(icon)
		name := agent.DisplayName()
		row := fmt.Sprintf(" %s  %s", iconStyled, name)

		if i == m.cursor {
			b.WriteString(selectedRowStyle.Width(width).Render(row))
		} else {
			b.WriteString(rowStyle.Width(width).Render(row))
		}
		if i < len(m.agents)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (m Model) renderDetail() string {
	if len(m.agents) == 0 || m.cursor >= len(m.agents) {
		return "No agent selected."
	}
	agent := m.agents[m.cursor]
	var state launchd.AgentState
	if m.cursor < len(m.states) {
		state = m.states[m.cursor]
	}

	return fmt.Sprintf(
		"[L] Logs   [I] Info\n\nAgent:  %s\nStatus: %s\nPID:    %s",
		agent.DisplayName(),
		launchd.StatusLabel(state.Status),
		pidStr(state.PID),
	)
}

func pidStr(pid int) string {
	if pid == 0 {
		return "—"
	}
	return fmt.Sprintf("%d", pid)
}
