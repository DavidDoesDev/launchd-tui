package ui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Leading-timestamp matcher. Covers the formats this project's agents emit:
//   2026-06-28T13:37:19      (ISO, local)
//   [2026-06-28T17:36:40Z]   (ISO, UTC, bracketed)
//   2026-06-05 00:14:08      (space-separated)
//   [15:44:24]               (time only — code-server)
var logTimestampRe = regexp.MustCompile(
	`^(\[?\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}Z?\]?|\[\d{2}:\d{2}:\d{2}\])\s*`,
)

var (
	logTimestampStyle = lipgloss.NewStyle().Foreground(ctpOverlay0)
	logErrorStyle     = lipgloss.NewStyle().Foreground(ctpRed)
	logWarnStyle      = lipgloss.NewStyle().Foreground(ctpYellow)
	logSuccessStyle   = lipgloss.NewStyle().Foreground(ctpGreen)
	logDefaultStyle   = lipgloss.NewStyle().Foreground(ctpSubtext0)
)

// Keyword sets are intentionally conservative — false positives in severity
// color are more distracting than a few uncolored lines. Note "disconnected"
// deliberately is NOT a success word.
var (
	errorWords   = []string{"error", "denied", "failed", "fail:", "timed out", "timeout", "unresponsive", "exception", "fatal", "panic", "permission denied", "refused", "no such"}
	warnWords    = []string{"warning", "warn:", "retry", "retrying", "unable", "not running", "deprecat"}
	successWords = []string{"done", "success", "succeeded", "pushed", "completed", "launching", "ready"}
)

func severityStyle(s string) lipgloss.Style {
	l := strings.ToLower(s)
	switch {
	case containsAny(l, errorWords):
		return logErrorStyle
	case containsAny(l, warnWords):
		return logWarnStyle
	case containsAny(l, successWords):
		return logSuccessStyle
	default:
		return logDefaultStyle
	}
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// styleLogLine dims a leading timestamp and colors the message by severity.
// Lines with no timestamp are treated as continuation/orphan output: indented
// two columns and colored by their own severity so errors still stand out.
func styleLogLine(line string) string {
	if strings.TrimSpace(line) == "" {
		return line
	}
	if loc := logTimestampRe.FindStringIndex(line); loc != nil {
		ts := line[:loc[1]]
		rest := line[loc[1]:]
		return logTimestampStyle.Render(ts) + severityStyle(rest).Render(rest)
	}
	return "  " + severityStyle(line).Render(line)
}

// styleLog applies styleLogLine across a whole buffer.
func styleLog(content string) string {
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = styleLogLine(l)
	}
	return strings.Join(lines, "\n")
}
