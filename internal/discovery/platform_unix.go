//go:build !windows

package discovery

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// GetClaudeProcesses discovers running Claude Code processes via ps.
func GetClaudeProcesses() []RawProc {
	out, err := exec.Command("ps", "-eo", "pid,ppid,%cpu,%mem,etime,command").Output()
	if err != nil {
		return nil
	}

	var procs []RawProc
	for i, line := range strings.Split(string(out), "\n") {
		if i == 0 { // skip header
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		cmd := strings.Join(fields[5:], " ")
		if !IsClaudeLine(cmd) {
			continue
		}

		pid, _ := strconv.Atoi(fields[0])
		ppid, _ := strconv.Atoi(fields[1])
		cpu, _ := strconv.ParseFloat(fields[2], 64)
		mem, _ := strconv.ParseFloat(fields[3], 64)

		procs = append(procs, RawProc{
			PID:     pid,
			PPID:    ppid,
			CPU:     cpu,
			Mem:     mem,
			Elapsed: fields[4],
			Command: cmd,
		})
	}
	return procs
}

// GetWorkDir resolves the working directory for a process.
// Uses /proc on Linux, lsof on macOS.
func GetWorkDir(pid int) string {
	if runtime.GOOS == "linux" {
		target, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
		if err == nil {
			return target
		}
		return ""
	}

	// macOS: fall back to lsof
	out, err := exec.Command("lsof", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 9 && fields[3] == "cwd" {
			return strings.Join(fields[8:], " ")
		}
	}
	return ""
}

// GetSource detects the IDE or terminal that launched a Claude session.
func GetSource(ppid int, cmd string) string {
	if ide := DetectIDEFromCmd(cmd); ide != "" {
		return ide
	}

	out, err := exec.Command("ps", "-p", strconv.Itoa(ppid), "-o", "comm=").Output()
	if err != nil {
		return "CLI"
	}
	parent := strings.ToLower(strings.TrimSpace(string(out)))

	switch {
	case strings.Contains(parent, "code"):
		return "VSCode"
	case strings.Contains(parent, "cursor"):
		return "Cursor"
	case strings.Contains(parent, "terminal"):
		return "Terminal"
	case strings.Contains(parent, "iterm"):
		return "iTerm"
	case strings.Contains(parent, "warp"):
		return "Warp"
	case strings.Contains(parent, "tmux"):
		return "tmux"
	case strings.Contains(parent, "alacritty"):
		return "Alacritty"
	case strings.Contains(parent, "kitty"):
		return "Kitty"
	case strings.Contains(parent, "konsole"):
		return "Konsole"
	case strings.Contains(parent, "gnome-terminal"):
		return "GNOME Term"
	case strings.Contains(parent, "ghostty"):
		return "Ghostty"
	case strings.Contains(parent, "wezterm"):
		return "WezTerm"
	case strings.Contains(parent, "hyper"):
		return "Hyper"
	case strings.Contains(parent, "foot"):
		return "foot"
	case strings.Contains(parent, "rio"):
		return "Rio"
	}
	return "CLI"
}
