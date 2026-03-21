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
	{"webstorm", "WebStorm"},
	{"goland", "GoLand"},
	{"intellij", "IntelliJ"},
	{"pycharm", "PyCharm"},
	{"phpstorm", "PhpStorm"},
	{"rubymine", "RubyMine"},
	{"rider", "Rider"},
	{"clion", "CLion"},
	{"datagrip", "DataGrip"},
}

// sourceToAppName maps Source display names to the macOS application name
// used in AppleScript and System Events. Only entries that differ are listed.
var sourceToAppName = map[string]string{
	"iTerm":      "iTerm2",
	"GNOME Term": "gnome-terminal",
}

// appNameFor returns the macOS application name for a given Source string.
func appNameFor(source string) string {
	if mapped, ok := sourceToAppName[source]; ok {
		return mapped
	}
	return source
}

// ideCLICommands maps IDE source names to their CLI command names.
// These IDEs are focused by opening the WorkDir via their CLI tool.
var ideCLICommands = map[string]string{
	"VSCode":    "code",
	"Cursor":    "cursor",
	"Windsurf":  "windsurf",
	"WebStorm":  "webstorm",
	"GoLand":    "goland",
	"IntelliJ":  "idea",
	"PyCharm":   "pycharm",
	"PhpStorm":  "phpstorm",
	"RubyMine":  "rubymine",
	"Rider":     "rider",
	"CLion":     "clion",
	"DataGrip":  "datagrip",
}

// FocusSession attempts to bring the terminal or IDE running the given
// session to the foreground. This is the sole public entry point.
//
// Flow:
//  1. IDE → launch IDE CLI command
//  2. tmux → select the matching pane
//  3. Terminal → findParentApp → discoverWindows → bestMatch → raiseWindow
func FocusSession(s discovery.Session) error {
	// IDE shortcut — CLI commands work reliably
	if cli, ok := ideCLICommands[s.Source]; ok {
		return exec.Command(cli, s.WorkDir).Start()
	}

	// tmux shortcut — has its own API
	if s.Source == "tmux" {
		return focusTmux(s.PPID)
	}

	// Terminal window focus via the discovery pipeline
	return focusTerminalWindow(s)
}

// focusTerminalWindow implements the new pipeline:
// findParentApp → discoverWindows → bestMatch → raiseWindow
func focusTerminalWindow(s discovery.Session) error {
	// Step 1: Identify the terminal application
	app := parentApp{name: s.Source}
	if s.Source == "CLI" {
		resolved, err := findParentApp(s.PPID)
		if err != nil {
			return fmt.Errorf("cannot identify terminal for session PID %d: %w", s.PID, err)
		}
		app = resolved
		if app.name == "tmux" {
			return focusTmux(s.PPID)
		}
		if cli, ok := ideCLICommands[app.name]; ok {
			return exec.Command(cli, s.WorkDir).Start()
		}
	}

	if runtime.GOOS != "darwin" {
		return fmt.Errorf("cannot focus %s on %s", app.name, runtime.GOOS)
	}

	// Step 2: Discover all windows for this terminal/IDE
	windows, err := discoverWindows(app, s.PPID)

	// Step 3: If only one window, raise it directly
	if err == nil && len(windows) == 1 {
		return raiseWindow(windows[0])
	}

	// Step 4: Multiple windows — pick the best match
	if err == nil && len(windows) > 1 {
		if best, ok := bestMatch(windows, s); ok {
			return raiseWindow(best)
		}
	}

	// Step 5: Fallback — just activate the app
	return activateMacApp(app.name)
}

// ---------------------------------------------------------------------------
// Process tree walking
// ---------------------------------------------------------------------------

// parentApp holds the result of a process tree walk: the display name
// and PID of the terminal/IDE application that owns a Claude session.
type parentApp struct {
	name string // display name, e.g. "WebStorm", "Ghostty", "Terminal"
	pid  int    // the actual PID of the app process
}

// findParentApp walks up the process tree from pid to find the terminal
// or IDE application. Returns the app's display name and PID.
func findParentApp(pid int) (parentApp, error) {
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
		cmdPath := strings.ToLower(line[spaceIdx+1:])

		// Check against known terminal patterns
		for _, p := range terminalPatterns {
			if strings.Contains(cmdPath, p.substring) {
				return parentApp{name: p.name, pid: current}, nil
			}
		}

		// Check for .app bundle path (macOS)
		if idx := strings.Index(line[spaceIdx+1:], ".app/"); idx >= 0 {
			prefix := line[spaceIdx+1 : spaceIdx+1+idx]
			name := prefix
			if slashIdx := strings.LastIndex(prefix, "/"); slashIdx >= 0 {
				name = prefix[slashIdx+1:]
			}
			return parentApp{name: name, pid: current}, nil
		}

		if nextPPID <= 1 {
			break
		}
		current = nextPPID
	}
	return parentApp{}, fmt.Errorf("no terminal found in process tree from PID %d", pid)
}

// ---------------------------------------------------------------------------
// Window discovery
// ---------------------------------------------------------------------------

// discoverWindows returns all windows for the given macOS application,
// enriched with titles, working directories, and TTY info where available.
func discoverWindows(app parentApp, sessionPPID int) ([]Window, error) {
	macName := appNameFor(app.name)
	switch macName {
	case "Terminal":
		return discoverTerminalAppWindows()
	case "iTerm2":
		return discoverITermWindows()
	default:
		return discoverGenericWindows(app, sessionPPID)
	}
}

// discoverTerminalAppWindows enumerates all Terminal.app windows and tabs,
// returning each tab as a Window with TTY and title.
func discoverTerminalAppWindows() ([]Window, error) {
	script := `
tell application "Terminal"
	set results to {}
	repeat with wi from 1 to count of windows
		set w to window wi
		repeat with ti from 1 to count of tabs of w
			set t to tab ti of w
			set tabTTY to tty of t
			set tabTitle to name of w
			set end of results to (wi as text) & "|" & (ti as text) & "|" & tabTTY & "|" & tabTitle
		end repeat
	end repeat
	set AppleScript's text item delimiters to linefeed
	return results as text
end tell`

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return nil, fmt.Errorf("Terminal.app AppleScript failed: %w", err)
	}

	var windows []Window
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		wi, _ := strconv.Atoi(parts[0])
		ti, _ := strconv.Atoi(parts[1])
		tty := strings.TrimPrefix(parts[2], "/dev/")
		title := parts[3]

		windows = append(windows, Window{
			AppName:  "Terminal",
			Title:    title,
			TTY:      tty,
			Index:    wi,
			TabIndex: ti,
		})
	}

	enrichWorkDirs(windows)

	return windows, nil
}

// discoverITermWindows enumerates all iTerm2 windows/tabs/sessions.
func discoverITermWindows() ([]Window, error) {
	script := `
tell application "iTerm2"
	set results to {}
	repeat with wi from 1 to count of windows
		set w to window wi
		repeat with ti from 1 to count of tabs of w
			set t to tab ti of w
			repeat with s in sessions of t
				set sTTY to tty of s
				set sTitle to name of s
				set end of results to (wi as text) & "|" & (ti as text) & "|" & sTTY & "|" & sTitle
			end repeat
		end repeat
	end repeat
	set AppleScript's text item delimiters to linefeed
	return results as text
end tell`

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return nil, fmt.Errorf("iTerm2 AppleScript failed: %w", err)
	}

	var windows []Window
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}
		wi, _ := strconv.Atoi(parts[0])
		ti, _ := strconv.Atoi(parts[1])
		tty := strings.TrimPrefix(parts[2], "/dev/")
		title := parts[3]

		windows = append(windows, Window{
			AppName:  "iTerm2",
			Title:    title,
			TTY:      tty,
			Index:    wi,
			TabIndex: ti,
		})
	}

	enrichWorkDirs(windows)

	return windows, nil
}

// discoverGenericWindows enumerates windows for a macOS app.
// Uses CGWindowListCopyWindowInfo (via Swift) which reliably sees all windows,
// then enriches with titles from System Events when available.
func discoverGenericWindows(app parentApp, sessionPPID int) ([]Window, error) {
	// Step 1: Get windows via CGWindowListCopyWindowInfo — sees all windows
	windows := discoverWindowsCG(app)

	// Step 2: Try to get titles from System Events (works for some apps)
	seTitles := discoverWindowsSE(app)

	// Merge: if CG windows lack titles but SE has them, use SE titles.
	// Match by index position since both enumerate in the same order.
	if len(seTitles) > 0 {
		for i := range windows {
			if windows[i].Title == "" && i < len(seTitles) {
				windows[i].Title = seTitles[i]
			}
		}
	}

	// Step 3: Resolve WorkDir from the session's shell process
	if sessionPPID > 0 {
		cwd := resolveWorkDir(sessionPPID)
		if cwd != "" {
			attached := false
			for i, w := range windows {
				if w.Title != "" && titleContainsPath(w.Title, cwd) {
					windows[i].WorkDir = cwd
					attached = true
				}
			}
			if !attached && len(windows) == 1 {
				windows[0].WorkDir = cwd
			}
		}
	}

	if len(windows) == 0 {
		return nil, fmt.Errorf("no windows found for %s (pid %d)", app.name, app.pid)
	}
	return windows, nil
}

// discoverWindowsCG uses CGWindowListCopyWindowInfo (via cgo on macOS)
// to enumerate all on-screen windows for a given PID. This is the most
// reliable method as it works for all apps including Java-based ones.
func discoverWindowsCG(app parentApp) []Window {
	if app.pid <= 0 {
		return nil
	}

	cgWindows := listCGWindows(app.pid)
	windows := make([]Window, 0, len(cgWindows))
	for i, cg := range cgWindows {
		windows = append(windows, Window{
			AppName: cg.Owner,
			Title:   cg.Title,
			Index:   i + 1,
		})
	}
	return windows
}

// discoverWindowsSE uses System Events to get window titles. This works for
// most apps but fails for some (e.g. Java-based apps when not frontmost).
// Returns just the titles in order, or nil on failure.
func discoverWindowsSE(app parentApp) []string {
	var processSelector string
	if app.pid > 0 {
		processSelector = fmt.Sprintf("first process whose unix id is %d", app.pid)
	} else {
		processSelector = fmt.Sprintf("first process whose name is \"%s\"", app.name)
	}

	script := fmt.Sprintf(`
tell application "System Events"
	set appProc to %s
	set results to {}
	repeat with w in windows of appProc
		set end of results to name of w
	end repeat
	set AppleScript's text item delimiters to linefeed
	return results as text
end tell`, processSelector)

	out, err := exec.Command("osascript", "-e", script).Output()
	if err != nil {
		return nil
	}

	text := strings.TrimSpace(string(out))
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

// titleContainsPath checks if a window title contains any component of the path.
func titleContainsPath(title, path string) bool {
	if title == "" || path == "" {
		return false
	}
	lower := strings.ToLower(title)
	if strings.Contains(lower, strings.ToLower(path)) {
		return true
	}
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		base := parts[len(parts)-1]
		if base == "" && len(parts) > 1 {
			base = parts[len(parts)-2]
		}
		return base != "" && strings.Contains(lower, strings.ToLower(base))
	}
	return false
}

// enrichWorkDirs resolves working directories for windows that have TTY info
// by finding the shell process on that TTY and reading its CWD.
func enrichWorkDirs(windows []Window) {
	if len(windows) == 0 {
		return
	}

	// Get all shell processes with their TTY and PID
	out, err := exec.Command("ps", "-eo", "pid,tty,comm").Output()
	if err != nil {
		return
	}

	// Build TTY → PID map for shell processes
	ttyToPID := make(map[string]int)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 3 {
			continue
		}
		comm := strings.ToLower(fields[2])
		if !isShell(comm) {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		tty := fields[1]
		if tty == "??" || tty == "?" {
			continue
		}
		ttyToPID[tty] = pid
	}

	// Resolve CWD for each window's TTY
	for i, w := range windows {
		if w.TTY == "" {
			continue
		}
		pid, ok := ttyToPID[w.TTY]
		if !ok {
			continue
		}
		cwd := resolveWorkDir(pid)
		if cwd != "" {
			windows[i].WorkDir = cwd
		}
	}
}

// isShell returns true if the process name is a common shell.
func isShell(name string) bool {
	base := name
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		base = name[idx+1:]
	}
	// Strip leading '-' (login shell convention)
	base = strings.TrimPrefix(base, "-")
	switch base {
	case "zsh", "bash", "fish", "sh", "dash", "tcsh", "csh", "nu", "nushell", "elvish", "xonsh":
		return true
	}
	return false
}

// resolveWorkDir gets the working directory for a process.
// Uses /proc on Linux, lsof on macOS.
func resolveWorkDir(pid int) string {
	return discovery.GetWorkDir(pid)
}

// ---------------------------------------------------------------------------
// Window raising
// ---------------------------------------------------------------------------

// raiseWindow brings a specific window to the foreground on macOS.
func raiseWindow(w Window) error {
	switch w.AppName {
	case "Terminal":
		return raiseTerminalWindow(w)
	case "iTerm2":
		return raiseITermWindow(w)
	default:
		return raiseGenericWindow(w)
	}
}

// raiseTerminalWindow focuses a specific Terminal.app window and tab.
func raiseTerminalWindow(w Window) error {
	if w.TTY != "" {
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
end tell`, w.TTY)
		if err := exec.Command("osascript", "-e", script).Run(); err == nil {
			return nil
		}
	}

	// Fall back to index-based raising
	if w.Index > 0 {
		script := fmt.Sprintf(`
tell application "Terminal"
	set index of window %d to 1
	if %d > 0 then
		set selected tab of window 1 to tab %d of window 1
	end if
	activate
end tell`, w.Index, w.TabIndex, w.TabIndex)
		return exec.Command("osascript", "-e", script).Run()
	}

	return activateMacApp("Terminal")
}

// raiseITermWindow focuses a specific iTerm2 window, tab, and session.
func raiseITermWindow(w Window) error {
	if w.TTY != "" {
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
end tell`, w.TTY)
		if err := exec.Command("osascript", "-e", script).Run(); err == nil {
			return nil
		}
	}

	return activateMacApp("iTerm2")
}

// raiseGenericWindow uses System Events to raise a window by index.
// Falls back to app-level activation if AXRaise fails (e.g. Java apps).
func raiseGenericWindow(w Window) error {
	if w.Index > 0 {
		script := fmt.Sprintf(`
tell application "System Events"
	set appProc to first process whose name is "%s"
	set targetWin to window %d of appProc
	perform action "AXRaise" of targetWin
	set frontmost of appProc to true
end tell`, w.AppName, w.Index)
		if err := exec.Command("osascript", "-e", script).Run(); err == nil {
			return nil
		}
	}
	return activateMacApp(w.AppName)
}

// ---------------------------------------------------------------------------
// tmux focus (preserved from original — works well)
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// Utilities
// ---------------------------------------------------------------------------

// activateMacApp performs a simple app-level activate via AppleScript.
func activateMacApp(name string) error {
	script := fmt.Sprintf(`tell application "%s" to activate`, name)
	return exec.Command("osascript", "-e", script).Run()
}
