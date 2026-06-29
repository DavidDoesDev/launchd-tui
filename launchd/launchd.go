package launchd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type Status int

const (
	StatusUnknown  Status = iota
	StatusRunning         // PID present
	StatusStopped         // no PID, exit 0
	StatusErrored         // no PID, non-zero exit
	StatusNotFound        // label not known to launchctl
)

type AgentState struct {
	Label    string
	Status   Status
	PID      int
	ExitCode int
	RunCount int
}

func GetState(label string) AgentState {
	out, err := exec.Command("launchctl", "list", label).Output()
	if err != nil {
		return AgentState{Label: label, Status: StatusNotFound}
	}

	state := AgentState{Label: label}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, `"PID"`) {
			if pid := parseValue(line); pid != "" && pid != "0" {
				state.PID, _ = strconv.Atoi(pid)
			}
		}
		if strings.HasPrefix(line, `"LastExitStatus"`) {
			if v := parseValue(line); v != "" {
				state.ExitCode, _ = strconv.Atoi(v)
			}
		}
	}

	if state.PID > 0 {
		state.Status = StatusRunning
	} else if state.ExitCode != 0 {
		state.Status = StatusErrored
	} else {
		state.Status = StatusStopped
	}

	return state
}

func parseValue(line string) string {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return ""
	}
	v := strings.TrimSpace(parts[1])
	v = strings.TrimSuffix(v, ";")
	v = strings.Trim(v, `"`)
	return strings.TrimSpace(v)
}

// LogPaths reads ~/Library/LaunchAgents/<label>.plist and returns
// StandardOutPath and StandardErrorPath. Missing keys return "".
func LogPaths(label string) (stdout, stderr string, err error) {
	plistPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", label+".plist")
	out, err := exec.Command("plutil", "-convert", "json", "-o", "-", plistPath).Output()
	if err != nil {
		return "", "", fmt.Errorf("plutil: %w", err)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(out, &data); err != nil {
		return "", "", fmt.Errorf("parse plist: %w", err)
	}
	if v, ok := data["StandardOutPath"].(string); ok {
		stdout = v
	}
	if v, ok := data["StandardErrorPath"].(string); ok {
		stderr = v
	}
	return stdout, stderr, nil
}

func StatusIcon(s Status) string {
	switch s {
	case StatusRunning:
		return "●"
	case StatusStopped:
		return "○"
	case StatusErrored:
		return "✗"
	case StatusNotFound:
		return "?"
	default:
		return "·"
	}
}

func StatusLabel(s Status) string {
	switch s {
	case StatusRunning:
		return "running"
	case StatusStopped:
		return "stopped"
	case StatusErrored:
		return "errored"
	case StatusNotFound:
		return "not found"
	default:
		return "unknown"
	}
}

// Schedule describes how/when an agent is launched, parsed from its plist.
type Schedule struct {
	Interval  int    // StartInterval seconds, 0 if none
	Calendar  string // human description of StartCalendarInterval, "" if none
	KeepAlive bool
	RunAtLoad bool
}

func GetSchedule(label string) Schedule {
	var sch Schedule
	plistPath := filepath.Join(os.Getenv("HOME"), "Library", "LaunchAgents", label+".plist")
	out, err := exec.Command("plutil", "-convert", "json", "-o", "-", plistPath).Output()
	if err != nil {
		return sch
	}
	var data map[string]interface{}
	if err := json.Unmarshal(out, &data); err != nil {
		return sch
	}
	if v, ok := data["StartInterval"].(float64); ok {
		sch.Interval = int(v)
	}
	if v, ok := data["RunAtLoad"].(bool); ok {
		sch.RunAtLoad = v
	}
	switch data["KeepAlive"].(type) {
	case bool:
		sch.KeepAlive = data["KeepAlive"].(bool)
	case map[string]interface{}:
		sch.KeepAlive = true // conditional keep-alive still means daemon-ish
	}
	if v, ok := data["StartCalendarInterval"]; ok {
		sch.Calendar = describeCalendar(v)
	}
	return sch
}

// Describe gives the agent's cadence in a short human form.
func (s Schedule) Describe() string {
	switch {
	case s.Interval > 0:
		return "every " + humanizeInterval(s.Interval)
	case s.Calendar != "":
		return s.Calendar
	case s.KeepAlive:
		return "kept alive"
	case s.RunAtLoad:
		return "at load"
	default:
		return "manual"
	}
}

func humanizeInterval(sec int) string {
	switch {
	case sec%3600 == 0:
		return fmt.Sprintf("%dh", sec/3600)
	case sec%60 == 0:
		return fmt.Sprintf("%dm", sec/60)
	default:
		return fmt.Sprintf("%ds", sec)
	}
}

func describeCalendar(v interface{}) string {
	switch cv := v.(type) {
	case map[string]interface{}:
		return describeCalDict(cv)
	case []interface{}:
		if len(cv) == 1 {
			if m, ok := cv[0].(map[string]interface{}); ok {
				return describeCalDict(m)
			}
		}
		return fmt.Sprintf("%d×/day", len(cv))
	}
	return "calendar"
}

func describeCalDict(m map[string]interface{}) string {
	get := func(k string) (int, bool) {
		if f, ok := m[k].(float64); ok {
			return int(f), true
		}
		return 0, false
	}
	hour, hOK := get("Hour")
	min, mOK := get("Minute")
	wd, wOK := get("Weekday")
	clock := ""
	if hOK || mOK {
		clock = fmt.Sprintf("%02d:%02d", hour, min)
	}
	switch {
	case wOK && clock != "":
		return weekdayName(wd) + " " + clock
	case clock != "":
		return "daily " + clock
	case wOK:
		return weekdayName(wd)
	default:
		return "calendar"
	}
}

func weekdayName(d int) string {
	names := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	return names[((d%7)+7)%7] // launchd treats 0 and 7 as Sunday
}

func Start(label string) error {
	return runLaunchctl("start", label)
}

func Stop(label string) error {
	return runLaunchctl("stop", label)
}

func runLaunchctl(cmd, label string) error {
	out, err := exec.Command("launchctl", cmd, label).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl %s %s: %w\n%s", cmd, label, err, out)
	}
	return nil
}
