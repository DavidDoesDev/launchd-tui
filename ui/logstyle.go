package ui

import (
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Severity keyword sets are intentionally conservative — false positives in
// color are more distracting than a few uncolored lines. Note "disconnected"
// deliberately is NOT a success word.
var (
	errorWords   = []string{"error", "denied", "failed", "fail:", "timed out", "timeout", "unresponsive", "exception", "fatal", "panic", "permission denied", "refused", "no such"}
	warnWords    = []string{"warning", "warn:", "retry", "retrying", "unable", "not running", "deprecat"}
	successWords = []string{"done", "success", "succeeded", "pushed", "completed", "launching", "ready"}
)

// Leading-timestamp matcher. Covers the formats this project's agents emit:
//
//	2026-06-28T13:37:19      (ISO, local)
//	[2026-06-28T17:36:40Z]   (ISO, UTC, bracketed)
//	2026-06-05 00:14:08      (space-separated)
//	[15:44:24]               (time only — code-server)
var logTimestampRe = regexp.MustCompile(
	`^(\[?\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}Z?\]?|\[\d{2}:\d{2}:\d{2}\])\s*`,
)

// severityClass ranks a line: 0 normal, 1 warn, 2 error. Used by both the line
// styler and the activity timeline so they agree on what "bad" looks like.
func severityClass(line string) int {
	l := strings.ToLower(line)
	switch {
	case containsAny(l, errorWords):
		return 2
	case containsAny(l, warnWords):
		return 1
	default:
		return 0
	}
}

func (s Styles) severityStyle(line string) lipgloss.Style {
	switch severityClass(line) {
	case 2:
		return s.logError
	case 1:
		return s.logWarn
	default:
		if containsAny(strings.ToLower(line), successWords) {
			return s.logSuccess
		}
		return s.logDefault
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
func (s Styles) styleLogLine(line string) string {
	if strings.TrimSpace(line) == "" {
		return line
	}
	if loc := logTimestampRe.FindStringIndex(line); loc != nil {
		ts := line[:loc[1]]
		rest := line[loc[1]:]
		return s.logTimestamp.Render(ts) + s.severityStyle(rest).Render(rest)
	}
	return "  " + s.severityStyle(line).Render(line)
}

// Datetime layouts tried against a line's leading timestamp token. Time-only
// formats are intentionally absent — without a date a line can't be bucketed,
// so it's treated as continuation output and inherits the prior line's bucket.
var logLayouts = []string{
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
}

// parseLogTime extracts an absolute time from a line's leading timestamp, or
// reports false when there's no date to anchor it.
func parseLogTime(line string) (time.Time, bool) {
	loc := logTimestampRe.FindStringIndex(line)
	if loc == nil {
		return time.Time{}, false
	}
	tok := strings.Trim(strings.TrimSpace(line[:loc[1]]), "[]")
	for _, layout := range logLayouts {
		if t, err := time.ParseInLocation(layout, tok, time.Local); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// lastTimestamp returns the most recent parseable timestamp in content.
func lastTimestamp(content string) (time.Time, bool) {
	lines := strings.Split(content, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if t, ok := parseLogTime(lines[i]); ok {
			return t, true
		}
	}
	return time.Time{}, false
}

// bucketFor maps a timestamp to a relative section. The live window (most
// recent 5 min) gets no header — it's the stream you're watching. Older entries
// split per calendar day with a date label.
func bucketFor(t, now time.Time) (id, label string, live bool) {
	lt := t.Local()
	switch d := now.Sub(t); {
	case d < 5*time.Minute:
		return "live", "", true
	case d < time.Hour:
		return "hour", "Last hour", false
	case sameDay(lt, now):
		return "today", "Earlier today", false
	case sameDay(lt, now.AddDate(0, 0, -1)):
		return "yesterday", "Yesterday", false
	case d < 7*24*time.Hour:
		return "week", "Earlier this week", false
	default:
		day := lt.Format("Jan 2")
		if lt.Year() != now.Year() {
			day = lt.Format("Jan 2, 2006")
		}
		return "older:" + day, day, false
	}
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// renderDivider draws a centered, uppercase section header flanked by heavy
// rules, e.g. ━━━━━━ YESTERDAY ━━━━━━.
func (s Styles) renderDivider(label string, width int) string {
	if width < 12 {
		width = 12
	}
	lbl := "  " + strings.ToUpper(label) + "  "
	side := (width - lipgloss.Width(lbl)) / 2
	if side < 2 {
		side = 2
	}
	right := width - side - lipgloss.Width(lbl)
	if right < 0 {
		right = 0
	}
	return s.logDividerRule.Render(strings.Repeat("━", side)) +
		s.logDividerLabel.Render(lbl) +
		s.logDividerRule.Render(strings.Repeat("━", right))
}

// styleLog styles each line and inserts a time-section divider whenever the
// bucket changes (except entering the live window). Lines without a parseable
// date inherit the running bucket, so dividers only appear where they're
// meaningful — a date-less log (e.g. time-only timestamps) gets none.
func (s Styles) styleLog(content string, width int) string {
	now := time.Now()
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	bucket := ""
	for _, line := range lines {
		if t, ok := parseLogTime(line); ok {
			if id, label, live := bucketFor(t, now); id != bucket {
				bucket = id
				if !live {
					// A full blank line on each side of the header.
					out = append(out, "", s.renderDivider(label, width), "")
				}
			}
		}
		out = append(out, s.styleLogLine(line))
	}
	return strings.Join(out, "\n")
}
