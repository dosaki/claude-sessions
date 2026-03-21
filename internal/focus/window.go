package focus

// Window represents a discovered OS-level window that may contain a Claude session.
type Window struct {
	AppName  string // display name: "Ghostty", "iTerm2", "Terminal", etc.
	Title    string // window title as reported by the OS
	WorkDir  string // working directory of the shell in this window (if resolvable)
	TTY      string // TTY device name, e.g. "ttys003" (Terminal.app/iTerm2 only)
	Index    int    // 1-based window index for raising (0 = unknown)
	TabIndex int    // 1-based tab index (-1 = not applicable)
	PaneID   string // tmux pane target: "session:window.pane"
}
