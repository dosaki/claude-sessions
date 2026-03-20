# claude-sessions

A dashboard that shows all running [Claude Code](https://docs.anthropic.com/en/docs/claude-code) sessions on your machine. Supports both a native GUI window (default) and a terminal-based dashboard.

```
 ◆ Claude Sessions Dashboard   2026-03-11 14:32:01  (refreshing every 2s · press Ctrl+C to exit)

  PID     STATUS               UPTIME  CPU%  MEM%  PROJECT                    SOURCE     AGENTS
  ─────────────────────────────────────────────────────────────────────────────────────────────────
  12345   ● Working             03:42   8.2   1.1  ~/dev/my-app               VSCode     2 running
  12400   ● Waiting for input   12:05   0.0   0.8  ~/dev/api-server           Terminal   none
  12510   ● Idle                01:15   0.0   0.5  ~/dev/docs                 CLI        none

  3 session(s)  ● 1 working  ● 1 waiting  ● 1 idle                           ↑/↓ navigate  Enter focus
```

## Features

- **GUI mode** (default) — native window powered by [Fyne](https://fyne.io) with clickable column headers, row click to focus, compact toggle, project filter dialog, and notification sounds
- **Terminal dashboard** (`--cli`) — ANSI-rendered live dashboard with keyboard navigation
- **One-shot** (`--once`) — print current sessions and exit
- Detects all running Claude Code processes (CLI, VSCode, Cursor, Windsurf)
- Shows session status: **Working**, **Waiting for input**, or **Idle**
- Displays PID, uptime, CPU%, MEM%, working directory, source, and subagent count
- **Session focusing** — click a row (GUI) or press Enter (CLI) to bring the session's terminal/IDE to the foreground
- **Sorting** — click column headers (GUI) or use `--sort` flag to sort by any column
- **Project filtering** — filter dialog (GUI) or `--filter` flag to show only specific projects
- **Compact mode** — `--compact` shows only status and project columns
- **Notification sounds** — optional alert when a session starts waiting for input (GUI toggle)
- Cross-platform: macOS, Linux, Windows

## Installation

### macOS / Linux (one-liner)

```bash
curl -fsSL https://raw.githubusercontent.com/dosaki/claude-sessions/main/install.sh | sh
```

This detects your OS and architecture, downloads the latest release, and installs it:
- **macOS** — installs `Claude Sessions.app` to `/Applications` (launchable from Spotlight/Launchpad) and symlinks the binary to `/usr/local/bin/claude-sessions`
- **Linux** — installs the binary to `/usr/local/bin/claude-sessions`

> **macOS Gatekeeper note:** Because the app is not signed with an Apple Developer certificate, macOS will block it on first launch. To allow it, run:
> ```bash
> xattr -dr com.apple.quarantine "/Applications/Claude Sessions.app"
> ```
> Or right-click the app in Finder → Open → click "Open" in the dialog.

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/dosaki/claude-sessions/main/install.ps1 | iex
```

Downloads the latest release and installs `claude-sessions.exe` to `~\.claude-sessions\bin`, adding it to your user `PATH`.

### Build from source

Requires Go 1.24+ and CGO (needed for Fyne's OpenGL bindings).

```bash
# Build using Make (recommended)
make build

# Or build directly
go build -o claude-sessions-bin ./cmd/claude-sessions

# macOS .app bundle
make app

# Optionally symlink into your PATH
ln -sf "$(pwd)/claude-sessions-bin" ~/bin/claude-sessions
```

## Usage

### GUI mode (default)

```bash
claude-sessions
```

Opens a native window with a session table, summary bar, compact toggle, notification toggle, and project filter button. Click a column header to sort. Click a row to focus that session's terminal or IDE.

### Terminal dashboard (`--cli`)

```bash
claude-sessions --cli
```

Live-updating ANSI dashboard in the terminal. Use keyboard shortcuts to navigate and focus sessions.

### One-shot (`--once`)

```bash
claude-sessions --once
```

Prints the current session table to stdout and exits. Useful for scripting.

### Examples

```bash
# Sort by CPU descending
claude-sessions --sort cpu --sort-desc

# Show only a specific project
claude-sessions --filter ~/dev/my-app

# Compact mode with 5s refresh
claude-sessions --cli --compact --interval 5

# Multiple project filters
claude-sessions --filter ~/dev/app1 --filter ~/dev/app2
```

## Options

| Flag              | Default | Description                                                            |
|-------------------|---------|------------------------------------------------------------------------|
| `--once`          | `false` | Print once and exit (terminal output)                                  |
| `--cli`           | `false` | Use terminal dashboard instead of GUI                                  |
| `--compact`       | `false` | Compact mode: show only status and project                             |
| `--interval N`    | `2`     | Refresh interval in seconds                                            |
| `--cpu-threshold` | `3.0`   | CPU% above which a session is considered "working"                     |
| `--sort COL`      | (none)  | Sort by column: pid, status, uptime, cpu, mem, project, source, agents |
| `--sort-desc`     | `false` | Sort in descending order (default is ascending)                        |
| `--filter PATH`   | (none)  | Only show sessions for this project path (repeatable)                  |

## Keyboard Shortcuts (CLI mode)

| Key        | Action                                      |
|------------|---------------------------------------------|
| `↑` / `k`  | Move selection up                           |
| `↓` / `j`  | Move selection down                         |
| `Enter`    | Focus the selected session's terminal/IDE   |
| `q`        | Quit                                        |

## Environment Variables

| Variable                    | Description                        |
|-----------------------------|------------------------------------|
| `CLAUDE_SESSIONS_INTERVAL`  | Default refresh interval (seconds) |

## How It Works

1. **Process discovery** — scans running processes for `claude` CLI instances, filtering out the desktop app, helpers, and this tool itself
2. **Subagent detection** — identifies child claude processes (subagents) by parent PID
3. **Working directory** — resolves each session's cwd via `lsof` (macOS), `/proc` (Linux), or PowerShell (Windows)
4. **Source detection** — determines the launch context (VSCode, Cursor, Terminal, etc.) from the command line and parent process
5. **Status classification** — uses CPU usage as the primary signal (above threshold = working), then reads the session's JSONL log to distinguish "waiting for input" (`tool_use` stop reason) from "idle"
6. **Sorting & filtering** — sessions are sorted by the selected column and filtered by project path
7. **Rendering** — GUI mode builds a Fyne table with clickable headers; CLI mode renders ANSI frames with cursor-home for flicker-free updates
8. **Session focusing** — on row click/Enter, activates the session's terminal or IDE (AppleScript on macOS, CLI commands for IDEs, tmux pane selection)

## Platform Support

| Platform | Process discovery | CWD resolution | CPU status      | Session focus       |
|----------|-------------------|----------------|-----------------|---------------------|
| macOS    | `ps`              | `lsof`         | Yes             | Terminal, IDE, tmux |
| Linux    | `ps`              | `/proc`        | Yes             | IDE, tmux           |
| Windows  | PowerShell        | PowerShell     | No (JSONL only) | IDE only            |
