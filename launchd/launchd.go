package launchd

import (
	"fmt"
	"os/exec"
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
}

func GetState(label string) AgentState {
	out, err := exec.Command("launchctl", "list", label).Output()
	if err != nil {
		if strings.Contains(err.Error(), "exit status 113") {
			return AgentState{Label: label, Status: StatusNotFound}
		}
		// non-113 error: treat as not found
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
	// lines look like: "PID" = 1234;
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return ""
	}
	v := strings.TrimSpace(parts[1])
	v = strings.TrimSuffix(v, ";")
	v = strings.Trim(v, `"`)
	return strings.TrimSpace(v)
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
