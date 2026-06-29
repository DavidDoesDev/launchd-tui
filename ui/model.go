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
const timelineHeight = 10 // activity chart rows atop the log view (incl. 2-row axis)

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
	vp             viewport.Model
	logContent     string
	autoScroll     bool
	logPath        string
	logOffset      int64
	tailGen        int
	styles         Styles
	settings       config.Settings
	showSettings   bool
	settingsCursor int
	showSwatches   bool // TEMP: swatch reference
	loadFrame      int  // animation frame for the loading placeholder
	animating      bool // a loading-animation ticker is in flight
	sparklePhase   int  // advances each sparkle tick to reshuffle the twinkle
}

type loadAnimMsg struct{}
type sparkleMsg struct{}

const loadAnimInterval = 90 * time.Millisecond
const sparkleInterval = 130 * time.Millisecond

func loadAnimCmd() tea.Cmd {
	return tea.Tick(loadAnimInterval, func(time.Time) tea.Msg { return loadAnimMsg{} })
}

func sparkleCmd() tea.Cmd {
	return tea.Tick(sparkleInterval, func(time.Time) tea.Msg { return sparkleMsg{} })
}

func New() Model {
	cfg, err := config.Load()
	settings := config.LoadSettings()

	theme := mochaTheme
	if t, ok := themeByName(settings.Theme); ok {
		theme = t
	}
	styles := newStyles(theme)
	sp := spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(styles.spinner),
	)
	return Model{
		agents:     cfg.Agents,
		states:     make([]launchd.AgentState, len(cfg.Agents)),
		loadErr:    err,
		activeTab:  tabLogs,
		autoScroll: true,
		spinner:    sp,
		actionIdx:  -1,
		styles:     styles,
		settings:   settings,
	}
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{fetchAllStates(m.agents), pollCmd(m.pollDuration()), sparkleCmd()}
	if m.settings.MouseWheel {
		cmds = append(cmds, tea.EnableMouseCellMotion)
	}
	return tea.Batch(cmds...)
}

func (m Model) pollDuration() time.Duration {
	if m.settings.PollIntervalSec > 0 {
		return time.Duration(m.settings.PollIntervalSec) * time.Second
	}
	return pollInterval
}

// --- commands ---

func pollCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return pollMsg{} })
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
	// Tails regardless of active tab so the activity timeline stays live on the
	// Info tab too.
	if len(m.agents) == 0 || m.cursor >= len(m.agents) {
		return nil
	}
	path := resolveLogPath(m.agents[m.cursor].Label)
	m.tailGen++
	m.logPath = path
	m.logOffset = 0
	m.logContent = ""
	m.autoScroll = true
	m.vp.SetContent("")

	cmds := []tea.Cmd{tailCmd(path, 0, m.tailGen)}
	// Kick the loading animation (once) while we wait for the first content.
	if !m.animating && path != "" {
		m.animating = true
		cmds = append(cmds, loadAnimCmd())
	}
	return tea.Batch(cmds...)
}

// --- update ---

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		_, _, _, _, rightContentW, rightContentH := m.layout()
		// Reserve rows in the right pane: timeline + 2 blanks + tab bar + 2 blanks.
		m.vp = viewport.New(rightContentW, rightContentH-timelineHeight-5)
		m.vp.SetContent(m.styles.styleLog(m.logContent, m.vp.Width))
		if m.autoScroll {
			m.vp.GotoBottom()
		}
		if cmd := m.startTail(); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case []launchd.AgentState:
		m.states = msg

	case sparkleMsg:
		m.sparklePhase++
		cmds = append(cmds, sparkleCmd())

	case loadAnimMsg:
		// Advance the loading wave only while still loading; otherwise let the
		// ticker die.
		if m.logContent == "" && m.logPath != "" {
			m.loadFrame++
			cmds = append(cmds, loadAnimCmd())
		} else {
			m.animating = false
		}

	case pollMsg:
		cmds = append(cmds, fetchAllStates(m.agents), pollCmd(m.pollDuration()))

	case tea.MouseMsg:
		// Wheel-scroll the log viewport; re-sync auto-scroll to whether we
		// ended up at the bottom.
		if m.activeTab == tabLogs {
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			cmds = append(cmds, cmd)
			m.autoScroll = m.vp.AtBottom()
		}

	case tailMsg:
		if msg.generation != m.tailGen {
			return m, nil // stale loop — discard
		}
		if msg.content != "" {
			m.logOffset += int64(len(msg.content))
			m.logContent += msg.content
			m.vp.SetContent(m.styles.styleLog(m.logContent, m.vp.Width))
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
		// While the settings modal is open it captures all keys.
		if m.showSettings {
			var cmd tea.Cmd
			m, cmd = m.handleSettingsKey(msg)
			return m, cmd
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case ",":
			m.showSettings = true
			return m, nil

		case "0": // TEMP: toggle swatch reference
			m.showSwatches = !m.showSwatches
			return m, nil

		case "esc":
			m.showSwatches = false
			return m, nil

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

// layout computes the Width()/Height() values for each pane. The left pane is
// borderless (total width == leftW). The right pane has a rounded border (+2 w,
// +2 h) and Padding(1,2) (+4 w, +2 h), so its Width()/Height() are reduced to
// keep both panes the same total size: leftW + rightW + 2 == m.width.
//
// rightContentW/rightContentH are the usable interior of the right pane (after
// its padding) for sizing the timeline and viewport.
func (m Model) layout() (leftW, leftH, rightW, rightH, rightContentW, rightContentH int) {
	panesH := m.height - 1 // bottom bar
	leftW = m.width * 30 / 100
	if leftW < 14 {
		leftW = 14
	}
	rightW = m.width - leftW - 2 // -2 for the right pane's border
	if rightW < 1 {
		rightW = 1
	}
	leftH = panesH
	rightH = panesH - 4 // border (2) + vertical padding (2)
	if rightH < 1 {
		rightH = 1
	}
	rightContentW = rightW - 4 // horizontal padding (2 each side)
	rightContentH = rightH
	return
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	leftW, leftH, rightW, rightH, rightContentW, _ := m.layout()

	left := m.styles.leftPane.
		Width(leftW).
		Height(leftH).
		Render(m.renderList(leftW - 2)) // -2 for the pane's horizontal padding

	right := m.styles.rightPane.
		Width(rightW).
		Height(rightH).
		Render(m.renderDetail(rightContentW))

	panes := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	bar := m.styles.bar.Width(m.width).Render("↑↓ navigate · s/x/r start/stop/restart · tab panel · , settings · q quit")

	view := lipgloss.JoinVertical(lipgloss.Left, panes, bar)

	if m.showSwatches { // TEMP
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center, m.renderSwatches(),
			lipgloss.WithWhitespaceChars(" "))
	}
	if m.showSettings {
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			m.renderSettings(),
			lipgloss.WithWhitespaceChars(" "),
		)
	}
	return view
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

	cards := make([]string, len(m.agents))
	for i, agent := range m.agents {
		var state launchd.AgentState
		if i < len(m.states) {
			state = m.states[i]
		}
		cards[i] = m.renderCard(i, agent, state, inner)
	}
	// Blank spacer line between cards.
	return "\n" + strings.Join(cards, "\n\n")
}

// renderCard draws one agent as a two-line card: line 1 is the status icon +
// name, line 2 an indented status line (state + pid/exit). The selected card
// fills with the herdr surface0 background and carries a fat left bar in the
// agent's status color; unselected cards reserve the same column with a blank.
// Every segment carries the background when selected so the fill has no
// SGR-reset holes.
func (m Model) renderCard(i int, agent config.Agent, state launchd.AgentState, inner int) string {
	selected := i == m.cursor
	var bg lipgloss.Color
	if selected {
		bg = m.styles.theme.Surface0 // herdr selected-tab navy
	}

	// Fat left bar (neutral) on the fill, or a blank reserving the column.
	barSeg := " "
	if selected {
		barSeg = lipgloss.NewStyle().Foreground(m.styles.theme.Subtext0).Background(bg).Render("▌")
	}

	var iconSeg string
	if i == m.actionIdx {
		iconSeg = m.spinner.View()
	} else {
		iconSeg = withBG(m.styles.statusIcon(state.Status), bg).Render(launchd.StatusIcon(state.Status))
	}
	nameFg := m.styles.theme.Subtext0
	if selected {
		nameFg = m.styles.selText
	}
	nameStyle := withBG(lipgloss.NewStyle().Foreground(nameFg).Bold(selected), bg)
	name := ansi.Truncate(agent.DisplayName(), inner-5, "…") // bar+gap+icon+gutter = 5

	line1 := barSeg + bgSpace(1, bg) + iconSeg + bgSpace(2, bg) + nameStyle.Render(name)
	line1 = padTo(line1, inner, bg)

	label, extra := statusLineText(state)
	line2 := barSeg + bgSpace(4, bg) + withBG(m.styles.statusLabel(state.Status), bg).Render(label)
	if extra != "" {
		line2 += withBG(m.styles.dim, bg).Render(" · " + extra)
	}
	line2 = padTo(line2, inner, bg)

	return line1 + "\n" + line2
}

// withBG applies a background when bg is non-empty.
func withBG(st lipgloss.Style, bg lipgloss.Color) lipgloss.Style {
	if bg == "" {
		return st
	}
	return st.Background(bg)
}

// bgSpace returns n spaces, filled with bg when bg is non-empty.
func bgSpace(n int, bg lipgloss.Color) string {
	s := strings.Repeat(" ", n)
	if bg == "" {
		return s
	}
	return lipgloss.NewStyle().Background(bg).Render(s)
}

// padTo right-pads an ANSI-containing line to inner columns, filling with bg.
func padTo(line string, inner int, bg lipgloss.Color) string {
	if w := lipgloss.Width(line); w < inner {
		return line + bgSpace(inner-w, bg)
	}
	return line
}

// placeholderBox centers a dim message in a w×h block, used to hold layout
// space while content is still loading.
func (m Model) placeholderBox(w, h int, msg string) string {
	if h < 1 {
		h = 1
	}
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, m.styles.dim.Render(msg))
}

// statusLineText returns the card's second-line label and optional detail.
func statusLineText(s launchd.AgentState) (label, extra string) {
	switch s.Status {
	case launchd.StatusRunning:
		return "running", fmt.Sprintf("pid %d", s.PID)
	case launchd.StatusStopped:
		return "stopped", ""
	case launchd.StatusErrored:
		return "errored", fmt.Sprintf("exit %d", s.ExitCode)
	case launchd.StatusNotFound:
		return "not loaded", ""
	default:
		return "—", ""
	}
}

func (m Model) renderDetail(contentW int) string {
	if len(m.agents) == 0 || m.cursor >= len(m.agents) {
		return "No agent selected."
	}

	// Activity timeline is always on top, regardless of the active tab. While the
	// log is still loading we render an animated full-height placeholder (with
	// the axis kept) so the layout holds its shape and nothing jumps.
	var timeline string
	switch {
	case m.logContent != "":
		timeline = m.styles.renderTimeline(m.logContent, contentW, timelineHeight, m.sparklePhase, m.settings.Animations)
	case m.logPath == "":
		timeline = m.placeholderBox(contentW, timelineHeight, "no activity")
	default:
		timeline = m.styles.renderLoadingTimeline(contentW, timelineHeight, m.loadFrame)
	}

	logsTab := m.styles.inactiveTab.Render("[L] Logs")
	infoTab := m.styles.inactiveTab.Render("[I] Info")
	if m.activeTab == tabLogs {
		logsTab = m.styles.activeTab.Render("[L] Logs")
	} else {
		infoTab = m.styles.activeTab.Render("[I] Info")
	}
	tabBar := logsTab + "   " + infoTab

	var body string
	switch m.activeTab {
	case tabLogs:
		switch {
		case m.logPath == "":
			body = m.placeholderBox(contentW, m.vp.Height, "no log file for this agent")
		case m.logContent == "":
			body = m.placeholderBox(contentW, m.vp.Height, "loading logs…")
		default:
			scrollHint := ""
			if !m.autoScroll {
				scrollHint = m.styles.dim.Render("  [scroll mode — G to resume]") + "\n"
			}
			body = scrollHint + m.vp.View()
		}

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
		for _, row := range rows {
			b.WriteString(m.styles.infoLabel.Render(row[0]) + "  " + m.styles.infoValue.Render(row[1]) + "\n")
		}
		body = strings.TrimRight(b.String(), "\n")
	}

	return timeline + "\n\n\n" + tabBar + "\n\n\n" + body
}

func pidStr(pid int) string {
	if pid == 0 {
		return "—"
	}
	return fmt.Sprintf("%d", pid)
}
