package ui

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/DavidDoesDev/launchd-tui/config"
	"github.com/DavidDoesDev/launchd-tui/launchd"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const pollInterval = 2 * time.Second
const logTailInterval = 500 * time.Millisecond
const pulseInterval = 800 * time.Millisecond

const (
	tabLogs = 0
	tabInfo = 1
)

// --- message types ---

type pollMsg struct{}

type tailMsg struct {
	content    string
	generation int
}

type pulseMsg struct{}

type actionDoneMsg struct {
	idx        int
	prevStatus launchd.Status
}

type actionPollMsg struct {
	idx        int
	prevStatus launchd.Status
}

type singleStateMsg struct {
	idx   int
	state launchd.AgentState
}

// --- model ---

type Model struct {
	width          int
	height         int
	agents         []config.Agent
	states         []launchd.AgentState
	cursor         int
	loadErr        error
	activeTab      int
	spinner        spinner.Model
	actionIdx      int
	actionDeadline time.Time
	pulsePhase     bool
	vp             viewport.Model
	logContent     string
	autoScroll     bool
	logPath        string
	logOffset      int64
	tailGen        int
}

func New() Model {
	cfg, err := config.Load()
	sp := spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(spinnerStyle),
	)
	return Model{
		agents:     cfg.Agents,
		states:     make([]launchd.AgentState, len(cfg.Agents)),
		loadErr:    err,
		activeTab:  tabLogs,
		autoScroll: true,
		spinner:    sp,
		actionIdx:  -1,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchAllStates(m.agents), pollCmd(), pulseCmd())
}

func pulseCmd() tea.Cmd {
	return tea.Tick(pulseInterval, func(time.Time) tea.Msg { return pulseMsg{} })
}

// --- commands ---

func pollCmd() tea.Cmd {
	return tea.Tick(pollInterval, func(time.Time) tea.Msg { return pollMsg{} })
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

func fetchOneState(agents []config.Agent, idx int) tea.Cmd {
	return func() tea.Msg {
		return singleStateMsg{idx: idx, state: launchd.GetState(agents[idx].Label)}
	}
}

func startCmd(label string, idx int, prev launchd.Status) tea.Cmd {
	return func() tea.Msg {
		launchd.Start(label) //nolint:errcheck
		return actionDoneMsg{idx: idx, prevStatus: prev}
	}
}

func stopCmd(label string, idx int, prev launchd.Status) tea.Cmd {
	return func() tea.Msg {
		launchd.Stop(label) //nolint:errcheck
		return actionDoneMsg{idx: idx, prevStatus: prev}
	}
}

func restartCmd(label string, idx int, prev launchd.Status) tea.Cmd {
	return func() tea.Msg {
		launchd.Stop(label)  //nolint:errcheck
		launchd.Start(label) //nolint:errcheck
		return actionDoneMsg{idx: idx, prevStatus: prev}
	}
}

func actionPollCmd(idx int, prev launchd.Status) tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg {
		return actionPollMsg{idx: idx, prevStatus: prev}
	})
}

func tailCmd(path string, offset int64, gen int) tea.Cmd {
	return tea.Tick(logTailInterval, func(time.Time) tea.Msg {
		if path == "" {
			return tailMsg{generation: gen}
		}
		f, err := os.Open(path)
		if err != nil {
			return tailMsg{generation: gen}
		}
		defer f.Close()
		if offset > 0 {
			if _, err := f.Seek(offset, io.SeekStart); err != nil {
				return tailMsg{generation: gen}
			}
		}
		b, err := io.ReadAll(f)
		if err != nil || len(b) == 0 {
			return tailMsg{generation: gen}
		}
		return tailMsg{content: string(b), generation: gen}
	})
}

func resolveLogPath(label string) string {
	stdout, stderr, err := launchd.LogPaths(label)
	if err != nil {
		return ""
	}
	if stdout != "" {
		return stdout
	}
	return stderr
}

func (m *Model) startTail() tea.Cmd {
	if m.activeTab != tabLogs || len(m.agents) == 0 || m.cursor >= len(m.agents) {
		return nil
	}
	path := resolveLogPath(m.agents[m.cursor].Label)
	m.tailGen++
	m.logPath = path
	m.logOffset = 0
	m.logContent = ""
	m.autoScroll = true
	m.vp.SetContent("")
	return tailCmd(path, 0, m.tailGen)
}

// --- update ---

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		_, rightW, contentHeight := m.layout()
		// viewport sits inside the right pane: -2 for pane padding,
		// -2 for the tab bar + its blank line.
		m.vp = viewport.New(rightW-2, contentHeight-2)
		m.vp.SetContent(m.logContent)
		if m.autoScroll {
			m.vp.GotoBottom()
		}
		if cmd := m.startTail(); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case []launchd.AgentState:
		m.states = msg

	case pulseMsg:
		m.pulsePhase = !m.pulsePhase
		cmds = append(cmds, pulseCmd())

	case pollMsg:
		cmds = append(cmds, fetchAllStates(m.agents), pollCmd())

	case tailMsg:
		if msg.generation != m.tailGen {
			return m, nil // stale loop — discard
		}
		if msg.content != "" {
			m.logOffset += int64(len(msg.content))
			m.logContent += msg.content
			m.vp.SetContent(m.logContent)
			if m.autoScroll {
				m.vp.GotoBottom()
			}
		}
		cmds = append(cmds, tailCmd(m.logPath, m.logOffset, m.tailGen))

	case singleStateMsg:
		if msg.idx < len(m.states) {
			m.states[msg.idx] = msg.state
		}

	case actionDoneMsg:
		return m, tea.Batch(fetchOneState(m.agents, msg.idx), actionPollCmd(msg.idx, msg.prevStatus))

	case actionPollMsg:
		statusChanged := msg.idx < len(m.states) && m.states[msg.idx].Status != msg.prevStatus
		if statusChanged || time.Now().After(m.actionDeadline) {
			m.actionIdx = -1
			return m, nil
		}
		return m, tea.Batch(fetchOneState(m.agents, msg.idx), actionPollCmd(msg.idx, msg.prevStatus))

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if cmd := m.startTail(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}

		case "down", "j":
			if m.cursor < len(m.agents)-1 {
				m.cursor++
				if cmd := m.startTail(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}

		case "pgup", "ctrl+u":
			if m.activeTab == tabLogs {
				var cmd tea.Cmd
				m.vp, cmd = m.vp.Update(msg)
				cmds = append(cmds, cmd)
				m.autoScroll = false
			}

		case "pgdown", "ctrl+d", "G":
			if m.activeTab == tabLogs {
				var cmd tea.Cmd
				m.vp, cmd = m.vp.Update(msg)
				cmds = append(cmds, cmd)
				if m.vp.AtBottom() {
					m.autoScroll = true
				}
			}

		case "tab":
			m.activeTab = (m.activeTab + 1) % 2
			if m.activeTab == tabLogs {
				if cmd := m.startTail(); cmd != nil {
					cmds = append(cmds, cmd)
				}
			}

		case "s":
			if len(m.agents) > 0 && m.actionIdx == -1 {
				idx := m.cursor
				prev := launchd.StatusUnknown
				if idx < len(m.states) {
					prev = m.states[idx].Status
				}
				m.actionIdx = idx
				m.actionDeadline = time.Now().Add(3 * time.Second)
				return m, tea.Batch(startCmd(m.agents[idx].Label, idx, prev), m.spinner.Tick)
			}

		case "x":
			if len(m.agents) > 0 && m.actionIdx == -1 {
				idx := m.cursor
				prev := launchd.StatusUnknown
				if idx < len(m.states) {
					prev = m.states[idx].Status
				}
				m.actionIdx = idx
				m.actionDeadline = time.Now().Add(3 * time.Second)
				return m, tea.Batch(stopCmd(m.agents[idx].Label, idx, prev), m.spinner.Tick)
			}

		case "r":
			if len(m.agents) > 0 && m.actionIdx == -1 {
				idx := m.cursor
				prev := launchd.StatusUnknown
				if idx < len(m.states) {
					prev = m.states[idx].Status
				}
				m.actionIdx = idx
				m.actionDeadline = time.Now().Add(3 * time.Second)
				return m, tea.Batch(restartCmd(m.agents[idx].Label, idx, prev), m.spinner.Tick)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// --- view ---

// layout computes pane widths and content height.
// A pane's total on-screen width is its Width() + 2 (rounded border, no margin).
// So leftW + rightW + 4 = m.width. Content height excludes the bottom bar (1)
// and the pane's top/bottom border (2).
func (m Model) layout() (leftW, rightW, contentHeight int) {
	leftW = m.width*30/100 - 2
	if leftW < 10 {
		leftW = 10
	}
	rightW = m.width - leftW - 4
	contentHeight = m.height - 1 - 2
	return
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	leftW, rightW, contentHeight := m.layout()

	left := leftPaneStyle.
		Width(leftW).
		Height(contentHeight).
		Render(m.renderList(leftW - 2)) // -2 for the pane's horizontal padding

	right := rightPaneStyle.
		Width(rightW).
		Height(contentHeight).
		Render(m.renderDetail())

	panes := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	bar := barStyle.Width(m.width).Render("↑↓ navigate · s start · x stop · r restart · tab panel · pgup/pgdn scroll logs · q quit")

	return lipgloss.JoinVertical(lipgloss.Left, panes, bar)
}

// renderList draws the agent list. `inner` is the exact column budget for each
// row (pane Width minus its horizontal padding). Every row is truncated to that
// budget BEFORE styling (lipgloss .Width wraps, it does not truncate), then
// rendered at .Width(inner) so selected and normal rows share an identical frame
// — this is what keeps the box from shifting when the selection moves.
func (m Model) renderList(inner int) string {
	if m.loadErr != nil {
		return fmt.Sprintf("error loading config:\n%v", m.loadErr)
	}
	if len(m.agents) == 0 {
		return "No agents configured.\n\nAdd entries to ~/.launchd-tui"
	}

	lines := make([]string, len(m.agents))
	for i, agent := range m.agents {
		var state launchd.AgentState
		if i < len(m.states) {
			state = m.states[i]
		}
		lines[i] = m.renderRow(i, agent, state, inner)
	}
	return strings.Join(lines, "\n")
}

// renderRow builds one list row as two background-carrying cells: a width-1
// icon cell and a width-(inner-1) name cell that includes the 2-space gutter.
// Rendering the cells separately (rather than embedding the pre-styled icon
// inside one Render) keeps the selection background continuous across the
// name text — a single Render would let the icon's SGR reset punch a hole in
// the highlight everywhere except the trailing padding.
func (m Model) renderRow(i int, agent config.Agent, state launchd.AgentState, inner int) string {
	selected := i == m.cursor

	iconStyle := statusIconStyle(state.Status, m.pulsePhase)
	nameStyle := rowStyle
	if selected {
		iconStyle = iconStyle.Background(ctpSurface0)
		nameStyle = selectedRowStyle
	}

	var iconSeg string
	if i == m.actionIdx {
		iconSeg = m.spinner.View()
	} else {
		iconSeg = iconStyle.Render(launchd.StatusIcon(state.Status))
	}

	name := ansi.Truncate(agent.DisplayName(), inner-3, "…") // -1 icon, -2 gutter
	nameSeg := nameStyle.Width(inner - 1).Render("  " + name)

	return iconSeg + nameSeg
}

func (m Model) renderDetail() string {
	if len(m.agents) == 0 || m.cursor >= len(m.agents) {
		return "No agent selected."
	}

	logsTab := inactiveTabStyle.Render("[L] Logs")
	infoTab := inactiveTabStyle.Render("[I] Info")
	if m.activeTab == tabLogs {
		logsTab = activeTabStyle.Render("[L] Logs")
	} else {
		infoTab = activeTabStyle.Render("[I] Info")
	}
	tabBar := logsTab + "   " + infoTab

	switch m.activeTab {
	case tabLogs:
		body := m.vp.View()
		if m.logPath == "" {
			body = dimStyle.Render("No log path configured in plist.")
		} else if m.logContent == "" {
			body = dimStyle.Render("(empty — waiting for output…)")
		}
		scrollHint := ""
		if !m.autoScroll {
			scrollHint = dimStyle.Render("  [scroll mode — G to resume]") + "\n"
		}
		return tabBar + "\n" + scrollHint + body

	default: // tabInfo
		agent := m.agents[m.cursor]
		var state launchd.AgentState
		if m.cursor < len(m.states) {
			state = m.states[m.cursor]
		}

		rows := [][2]string{
			{"Label", agent.Label},
			{"Name", agent.DisplayName()},
			{"Status", launchd.StatusLabel(state.Status)},
			{"PID", pidStr(state.PID)},
			{"Last exit", fmt.Sprintf("%d", state.ExitCode)},
			{"Run count", fmt.Sprintf("%d", state.RunCount)},
			{"Plist", fmt.Sprintf("~/Library/LaunchAgents/%s.plist", agent.Label)},
		}

		var b strings.Builder
		b.WriteString(tabBar + "\n\n")
		for _, row := range rows {
			b.WriteString(infoLabelStyle.Render(row[0]) + "  " + infoValueStyle.Render(row[1]) + "\n")
		}
		return strings.TrimRight(b.String(), "\n")
	}
}

func pidStr(pid int) string {
	if pid == 0 {
		return "—"
	}
	return fmt.Sprintf("%d", pid)
}
