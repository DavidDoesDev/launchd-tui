package ui

import (
	"testing"
	"time"
)

func TestParseLogTime(t *testing.T) {
	cases := []struct {
		line string
		ok   bool
	}{
		{"2026-06-28T13:37:19  NAS share unresponsive", true},
		{"[2026-06-28T17:36:40Z] sync-tasks: done", true},
		{"2026-06-05 00:14:08 [tailscale-keepalive] launching", true},
		{"[15:44:24] [100.116.47.89] disconnected", false}, // time only — no date
		{"mkdir: /Volumes/Ingest: Permission denied", false},
		{"", false},
	}
	for _, c := range cases {
		if _, ok := parseLogTime(c.line); ok != c.ok {
			t.Errorf("parseLogTime(%q) ok=%v, want %v", c.line, ok, c.ok)
		}
	}
}

func TestBucketFor(t *testing.T) {
	now := time.Date(2026, 6, 28, 18, 0, 0, 0, time.Local)
	cases := []struct {
		name      string
		t         time.Time
		wantID    string
		wantLive  bool
		wantLabel string
	}{
		{"live", now.Add(-2 * time.Minute), "live", true, ""},
		{"last hour", now.Add(-30 * time.Minute), "hour", false, "Last hour"},
		{"earlier today", now.Add(-5 * time.Hour), "today", false, "Earlier today"},
		{"yesterday", now.AddDate(0, 0, -1), "yesterday", false, "Yesterday"},
		{"this week", now.AddDate(0, 0, -4), "week", false, "Earlier this week"},
		{"older", now.AddDate(0, 0, -20), "older:Jun 8", false, "Jun 8"},
	}
	for _, c := range cases {
		id, label, live := bucketFor(c.t, now)
		if id != c.wantID || live != c.wantLive || label != c.wantLabel {
			t.Errorf("%s: bucketFor = (%q,%q,%v), want (%q,%q,%v)",
				c.name, id, label, live, c.wantID, c.wantLabel, c.wantLive)
		}
	}
}
