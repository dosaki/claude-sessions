// Package render formats and outputs the Claude Sessions dashboard.
package render

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"claude-sessions/internal/ansi"
	"claude-sessions/internal/config"
	"claude-sessions/internal/discovery"
	"claude-sessions/internal/terminal"
	"claude-sessions/internal/util"
)

// Layout constants for column widths.
const (
	fixedColumnsWidth = 80 // sum of fixed column widths (PID+STATUS+UPTIME+CPU+MEM+SOURCE+AGENTS+padding)
	minProjectWidth   = 12
	maxProjectWidth   = 48
	separatorMargin   = 4
	minSeparatorWidth = 40
	verticalMargin    = 7 // header + column headers + footer + margins

	compactFixedWidth  = 24 // status column + padding in compact mode
	compactMinProjW    = 10
	compactMaxProjW    = 60
	compactVertMargin  = 5 // header + column header + footer + margins
)

// shortenPath replaces the home directory prefix with "~".
var shortenPath = util.ShortenPath

func formatStatusDisplay(status string) string {
	switch status {
	case "working":
		return ansi.Blue + "●" + ansi.Reset + " " + ansi.Bold + "Working" + ansi.Reset
	case "waiting":
		return ansi.Yellow + "●" + ansi.Reset + " " + ansi.Bold + "Waiting for input" + ansi.Reset
	default:
		return ansi.Dim + "●" + ansi.Reset + " " + ansi.Dim + "Idle" + ansi.Reset
	}
}

func formatProject(workDir string, projectWidth int) string {
	project := shortenPath(workDir)
	if project == "" {
		return ansi.Dim + "unknown" + ansi.Reset
	}
	if ansi.VisibleLen(project) > projectWidth {
		project = util.TruncateProject(ansi.Re.ReplaceAllString(project, ""), projectWidth)
	}
	return project
}

func formatAgents(count int) string {
	if count > 0 {
		return ansi.Cyan + strconv.Itoa(count) + " running" + ansi.Reset
	}
	return ansi.Dim + "none" + ansi.Reset
}

func formatPercent(val float64) string {
	s := util.FormatPercent(val)
	if val == 0 {
		return ansi.Dim + s + ansi.Reset
	}
	return s
}

func sessionRow(s discovery.Session, projectWidth int) string {
	return fmt.Sprintf("  %s %s %s %s %s %s %s %s",
		ansi.Pad(ansi.Bold+strconv.Itoa(s.PID)+ansi.Reset, 7),
		ansi.Pad(formatStatusDisplay(s.Status), 20),
		ansi.Pad(s.Elapsed, 12),
		ansi.Pad(formatPercent(s.CPU), 7),
		ansi.Pad(formatPercent(s.Mem), 7),
		ansi.Pad(formatProject(s.WorkDir, projectWidth), projectWidth),
		ansi.Pad(ansi.Dim+s.Source+ansi.Reset, 10),
		formatAgents(s.SubagentCount),
	)
}

func compactRow(s discovery.Session, projectWidth int) string {
	return fmt.Sprintf(" %s %s",
		ansi.Pad(formatStatusDisplay(s.Status), 20),
		formatProject(s.WorkDir, projectWidth),
	)
}

func summaryLine(total, working, waiting, idle int) string {
	s := fmt.Sprintf("  %s%d%s session(s)", ansi.Bold, total, ansi.Reset)
	if working > 0 {
		s += fmt.Sprintf("  %s●%s %d working", ansi.Blue, ansi.Reset, working)
	}
	if waiting > 0 {
		s += fmt.Sprintf("  %s●%s %d waiting", ansi.Yellow, ansi.Reset, waiting)
	}
	if idle > 0 {
		s += fmt.Sprintf("  %s●%s %d idle", ansi.Dim, ansi.Reset, idle)
	}
	return s
}

// columnSortName maps display header names to sort column keys.
var columnSortName = map[string]string{
	"PID": "pid", "STATUS": "status", "UPTIME": "uptime",
	"CPU%": "cpu", "MEM%": "mem", "PROJECT": "project",
	"SOURCE": "source", "AGENTS": "agents",
}

// sortIndicator returns ▲ or ▼ if header matches the active sort column.
func sortIndicator(header string, sortCol string, asc bool) string {
	if columnSortName[header] == sortCol {
		if asc {
			return " ▲"
		}
		return " ▼"
	}
	return ""
}

// Draw renders one frame of the dashboard to stdout.
// sessions is the pre-fetched, filtered, and sorted session list.
// selectedRow is the highlighted row index (-1 for no selection).
func Draw(cfg config.Config, prevLines *int, sessions []discovery.Session, selectedRow int, statusMsg string) {
	cols, rows := terminal.GetSize()
	now := time.Now().Format("2006-01-02 15:04:05")

	var buf strings.Builder
	lineCount := 0

	emit := func(s string) {
		buf.WriteString(ansi.Truncate(s, cols))
		buf.WriteString(ansi.ClearEOL)
		buf.WriteString("\n")
		lineCount++
	}

	interactive := !cfg.Once
	compact := cfg.Compact
	var projectWidth, sepWidth, vMargin int
	if compact {
		projectWidth = ansi.Clamp(cols-compactFixedWidth, compactMinProjW, compactMaxProjW)
		sepWidth = ansi.Clamp(cols-separatorMargin, minSeparatorWidth, cols)
		vMargin = compactVertMargin
	} else {
		projectWidth = ansi.Clamp(cols-fixedColumnsWidth, minProjectWidth, maxProjectWidth)
		sepWidth = ansi.Clamp(cols-separatorMargin, minSeparatorWidth, cols)
		vMargin = verticalMargin
	}
	if interactive {
		vMargin++ // hint bar
	}

	// Header
	if compact {
		header := ansi.Bold + " ◆ Sessions" + ansi.Reset
		header += ansi.Dim + "  " + now + ansi.Reset
		emit(header)
	} else {
		header := ansi.Bold + ansi.BgBlue + ansi.White + " ◆ Claude Sessions Dashboard " + ansi.Reset
		header += ansi.Dim + "  " + now + "  " + ansi.Reset
		if !cfg.Once {
			header += ansi.Dim + fmt.Sprintf("(refreshing every %s)", cfg.Interval) + ansi.Reset
		}
		emit(header)
		emit("")
	}

	if len(sessions) == 0 {
		if !compact {
			emit("  " + ansi.Dim + "No active Claude sessions found." + ansi.Reset)
			emit("")
		}
		emit("  " + ansi.Dim + "Start a session with: " + ansi.Reset + ansi.Bold + "claude" + ansi.Reset)
	} else {
		if compact {
			emit(" " + ansi.Dim + strings.Repeat("─", sepWidth) + ansi.Reset)
		} else {
			headers := [8]string{"PID", "STATUS", "UPTIME", "CPU%", "MEM%", "PROJECT", "SOURCE", "AGENTS"}
			for i := range headers {
				headers[i] += sortIndicator(headers[i], cfg.SortColumn, cfg.SortAsc)
			}
			emit(fmt.Sprintf("  "+ansi.Dim+"%-7s %-20s %-12s %-7s %-7s %-"+strconv.Itoa(projectWidth)+"s %-10s %s"+ansi.Reset,
				headers[0], headers[1], headers[2], headers[3], headers[4], headers[5], headers[6], headers[7]))
			emit("  " + ansi.Dim + strings.Repeat("─", sepWidth) + ansi.Reset)
		}

		maxRows := ansi.Clamp(rows-vMargin, 1, len(sessions))

		// Compute scroll offset so the selected row is always visible.
		scrollOffset := 0
		if selectedRow >= maxRows {
			scrollOffset = selectedRow - maxRows + 1
		}

		// Count statuses across ALL sessions (not just visible window).
		working, waiting, idle := 0, 0, 0
		for _, s := range sessions {
			switch s.Status {
			case "working":
				working++
			case "waiting":
				waiting++
			default:
				idle++
			}
		}

		end := scrollOffset + maxRows
		if end > len(sessions) {
			end = len(sessions)
		}

		for i := scrollOffset; i < end; i++ {
			s := sessions[i]
			var row string
			if compact {
				row = compactRow(s, projectWidth)
			} else {
				row = sessionRow(s, projectWidth)
			}
			if i == selectedRow {
				row = ansi.Reverse + row + ansi.Reset
			}
			emit(row)
		}

		if len(sessions) > end {
			emit(fmt.Sprintf("  "+ansi.Dim+"… and %d more"+ansi.Reset, len(sessions)-end))
		}

		if !compact {
			emit("")
		}
		emit(summaryLine(len(sessions), working, waiting, idle))
	}

	// Hint bar for interactive mode
	if interactive {
		emit("  " + ansi.Dim + "↑↓/jk Navigate  Enter Focus session  q Quit" + ansi.Reset)
		if statusMsg != "" {
			emit("  " + ansi.Yellow + "⚠ " + statusMsg + ansi.Reset)
		}
	}

	// Flush
	if !cfg.Once {
		fmt.Print(ansi.CursorHome)
	}
	fmt.Print(buf.String())

	// Clear leftover lines from previous frame
	if !cfg.Once && *prevLines > lineCount {
		for range *prevLines - lineCount {
			fmt.Print(ansi.ClearEOL + "\n")
		}
	}
	*prevLines = lineCount
}
