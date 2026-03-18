//go:build !windows

package focus

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"claude-sessions/internal/discovery"
)

// terminalPatterns maps process name substrings to display names for known
// terminal emulators. Checked in order during process tree walks.
var terminalPatterns = []struct {
	substring string
	name      string
}{
	{"tmux", "tmux"},
	{"terminal", "Terminal"},
	{"iterm", "iTerm"},
	{"warp", "Warp"},
	{"alacritty", "Alacritty"},
	{"kitty", "Kitty"},
	{"konsole", "Konsole"},
	{"gnome-terminal", "GNOME Term"},
	{"ghostty", "Ghostty"},
	{"wezterm", "WezTerm"},
	{"hyper", "Hyper"},
	{"foot", "foot"},
	{"rio", "Rio"},
}

// FocusSession attempts to bring the terminal or IDE running the given
// session to the foreground. Best-effort: returns an error if the source
// type is not supported on this platform.
func FocusSession(s discovery.Session) error {
	switch s.Source {
	case "tmux":
		return focusTmux(s.PPID)
	case "VSCode":
		return exec.Command("code", s.WorkDir).Start()
	case "Cursor":
		return exec.Command("cursor", s.WorkDir).Start()
	case "Windsurf":
		return exec.Command("windsurf", s.WorkDir).Start()
	case "Terminal", "iTerm", "Warp", "Alacritty", "Kitty",
		"GNOME Term", "Konsole", "Ghostty", "WezTerm", "Hyper",
		"foot", "Rio":
		if runtime.GOOS == "darwin" {
			return focusMacApp(s)
		}
		return fmt.Errorf("cannot focus %s on %s", s.Source, runtime.GOOS)
	case "CLI":
		return focusCLI(s)
	default:
		return fmt.Errorf("cannot focus %q sessions", s.Source)
	}
}

// focusCLI walks up the process tree from the session's parent to find the
// terminal emulator, then activates it. This handles the common case where
// GetSource returns "CLI" because the immediate parent is a shell (zsh, bash)
// rather than the terminal app itself.
//
// On macOS, if the tree walk fails to identify the terminal, it falls back
// to TTY-based window matching in Terminal.app and iTerm2.
func focusCLI(s discovery.Session) error {
	name, err := findTerminalInTree(s.PPID)
	if err == nil {
		if name == "tmux" {
			return focusTmux(s.PPID)
		}
		if runtime.GOOS == "darwin" {
			resolved := s
			resolved.Source = name
			return focusMacApp(resolved)
		}
		return fmt.Errorf("cannot focus %s on %s", name, runtime.GOOS)
	}

	// Tree walk failed — try TTY-based fallback on macOS.
	if runtime.GOOS == "darwin" {
		return focusMacCLIByTTY(s)
	}
	return err
}

// focusMacCLIByTTY attempts to focus a CLI session on macOS by resolving its
// TTY and trying Terminal.app and iTerm2 window matching. If neither has a
// matching window, it activates whichever terminal app is currently frontmost.
func focusMacCLIByTTY(s discovery.Session) error {
	tty := getTTY(s.PPID)
	if tty != "" {
		// Try Terminal.app first, then iTerm2 — these expose TTY via AppleScript.
		if err := focusMacTerminalWindow(tty); err == nil {
			return nil
		}
		if err := focusMacITermWindow(tty); err == nil {
			return nil
		}
	}
	// Try window-title matching if we can find the terminal app.
	appName := findMacAppInTree(s.PPID)
	if appName != "" {
		if s.WorkDir != "" {
			if err := focusMacWindowByTitle(appName, s.WorkDir); err == nil {
				return nil
			}
		}
		return activateMacApp(appName)
	}
	return fmt.Errorf("could not find terminal app for CLI session (PID %d)", s.PPID)
}

// findMacAppInTree walks up the process tree looking for a macOS .app bundle
// process (indicated by a path containing ".app/"). Returns the app name
// suitable for AppleScript activation, or "" if not found.
func findMacAppInTree(pid int) string {
	current := pid
	for range 10 {
		out, err := exec.Command("ps", "-p", strconv.Itoa(current), "-o", "ppid=,command=").Output()
		if err != nil {
			break
		}
		line := strings.TrimSpace(string(out))
		spaceIdx := strings.IndexByte(line, ' ')
		if spaceIdx < 0 {
			break
		}
		nextPPID, err := strconv.Atoi(line[:spaceIdx])
		if err != nil {
			break
		}
		cmdPath := line[spaceIdx+1:]

		// Look for .app bundle path, e.g. /Applications/Ghostty.app/Contents/MacOS/ghostty
		if idx := strings.Index(cmdPath, ".app/"); idx >= 0 {
			// Extract the app name from the path
			prefix := cmdPath[:idx]
			slashIdx := strings.LastIndex(prefix, "/")
			if slashIdx >= 0 {
				return prefix[slashIdx+1:]
			}
			return prefix
		}

		if nextPPID <= 1 {
			break
		}
		current = nextPPID
	}
	return ""
}

// findTerminalInTree walks up the process tree from pid, checking each
// ancestor's name against known terminal emulators. Returns the terminal
// name or an error if none is found within 10 levels.
func findTerminalInTree(pid int) (string, error) {
	current := pid
	for range 10 {
		out, err := exec.Command("ps", "-p", strconv.Itoa(current), "-o", "ppid=,comm=").Output()
		if err != nil {
			break
		}
		fields := strings.Fields(strings.TrimSpace(string(out)))
		if len(fields) < 2 {
			break
		}
		nextPPID, err := strconv.Atoi(fields[0])
		if err != nil {
			break
		}
		comm := strings.ToLower(strings.Join(fields[1:], " "))

		for _, p := range terminalPatterns {
			if strings.Contains(comm, p.substring) {
				return p.name, nil
			}
		}

		if nextPPID <= 1 {
			break
		}
		current = nextPPID
	}
	return "", fmt.Errorf("could not find terminal app for CLI session (PID %d)", pid)
}

// focusTmux finds the tmux pane whose PID matches the session's parent
// process, then switches to that window and pane.
func focusTmux(ppid int) error {
	out, err := exec.Command("tmux", "list-panes", "-a",
		"-F", "#{pane_pid} #{session_name}:#{window_index}.#{pane_index}").Output()
	if err != nil {
		return fmt.Errorf("tmux not available: %w", err)
	}
	ppidStr := strconv.Itoa(ppid)
	var target string
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == ppidStr {
			target = fields[1]
			break
		}
	}
	if target == "" {
		return fmt.Errorf("tmux pane not found for PPID %d", ppid)
	}
	if err := exec.Command("tmux", "select-window", "-t", target).Run(); err != nil {
		return err
	}
	return exec.Command("tmux", "select-pane", "-t", target).Run()
}

// getTTY returns the TTY device name (e.g. "ttys003") for the given PID,
// or "" if it cannot be determined.
func getTTY(pid int) string {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "tty=").Output()
	if err != nil {
		return ""
	}
	tty := strings.TrimSpace(string(out))
	if tty == "" || tty == "??" {
		return ""
	}
	return tty
}

// activateMacApp performs a simple app-level activate via AppleScript.
// This brings the application to the foreground but does not select a
// specific window or tab.
func activateMacApp(name string) error {
	script := fmt.Sprintf(`tell application "%s" to activate`, name)
	return exec.Command("osascript", "-e", script).Run()
}

// focusMacApp attempts to focus the specific terminal window/tab containing
// the session. For Terminal.app and iTerm2, it resolves the session's TTY
// and uses AppleScript to select the matching window and tab. For other
// terminals, it falls back to app-level activation.
func focusMacApp(s discovery.Session) error {
	name := s.Source
	if name == "iTerm" {
		name = "iTerm2"
	}

	tty := getTTY(s.PPID)
	if tty != "" {
		switch s.Source {
		case "Terminal":
			if err := focusMacTerminalWindow(tty); err == nil {
				return nil
			}
		case "iTerm":
			if err := focusMacITermWindow(tty); err == nil {
				return nil
			}
		}
	}

	// TTY matching failed or not available — try matching by window title.
	// Most terminals include the CWD or running command in the window title.
	if s.WorkDir != "" {
		if err := focusMacWindowByTitle(name, s.WorkDir); err == nil {
			return nil
		}
	}

	return activateMacApp(name)
}

// focusMacWindowByTitle uses AppleScript via System Events to find a window
// of the given app whose title contains the search string (typically the
// session's working directory). This works for any macOS app that exposes
// standard window names and serves as a fallback when TTY matching is not
// available (e.g. Ghostty, Alacritty, Kitty, WezTerm).
func focusMacWindowByTitle(appName, search string) error {
	// Try matching the full path first, then just the basename for shorter
	// window titles (e.g. "~/project" vs "/Users/x/project").
	parts := strings.Split(search, "/")
	basename := parts[len(parts)-1]
	if basename == "" && len(parts) > 1 {
		basename = parts[len(parts)-2]
	}

	// Use System Events to enumerate windows — this works generically across
	// all apps without needing their specific scripting dictionary.
	script := fmt.Sprintf(`
tell application "System Events"
	set appProc to first process whose name is "%s"
	repeat with w in windows of appProc
		set winName to name of w
		if winName contains "%s" or winName contains "%s" then
			perform action "AXRaise" of w
			set frontmost of appProc to true
			return
		end if
	end repeat
	error "no matching window"
end tell`, appName, search, basename)
	return exec.Command("osascript", "-e", script).Run()
}

// focusMacTerminalWindow uses AppleScript to find and focus the Terminal.app
// window and tab whose TTY matches the given tty name (e.g. "ttys003").
func focusMacTerminalWindow(tty string) error {
	// Terminal.app exposes `tty` on each tab as a full path like "/dev/ttys003".
	script := fmt.Sprintf(`
tell application "Terminal"
	set targetTTY to "/dev/%s"
	repeat with w in windows
		repeat with t in tabs of w
			if tty of t is targetTTY then
				set index of w to 1
				set selected tab of w to t
				activate
				return
			end if
		end repeat
	end repeat
	error "TTY not found"
end tell`, tty)
	return exec.Command("osascript", "-e", script).Run()
}

// focusMacITermWindow uses AppleScript to find and focus the iTerm2 window,
// tab, and session whose TTY matches the given tty name (e.g. "ttys003").
func focusMacITermWindow(tty string) error {
	// iTerm2 exposes `tty` on each session as a full path like "/dev/ttys003".
	script := fmt.Sprintf(`
tell application "iTerm2"
	set targetTTY to "/dev/%s"
	repeat with w in windows
		repeat with t in tabs of w
			repeat with s in sessions of t
				if tty of s is targetTTY then
					select w
					select t
					select s
					activate
					return
				end if
			end repeat
		end repeat
	end repeat
	error "TTY not found"
end tell`, tty)
	return exec.Command("osascript", "-e", script).Run()
}
