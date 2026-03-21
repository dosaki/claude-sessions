package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"time"

	"claude-sessions/internal/ansi"
	"claude-sessions/internal/config"
	"claude-sessions/internal/discovery"
	"claude-sessions/internal/focus"
	"claude-sessions/internal/gui"
	"claude-sessions/internal/render"
	"claude-sessions/internal/terminal"
)

// stringSliceFlag collects repeated flag values into a slice.
type stringSliceFlag []string

func (f *stringSliceFlag) String() string { return fmt.Sprintf("%v", *f) }
func (f *stringSliceFlag) Set(val string) error {
	*f = append(*f, val)
	return nil
}

// fetchSessions discovers, filters, and sorts Claude sessions.
func fetchSessions(cfg config.Config) []discovery.Session {
	procs := discovery.GetClaudeProcesses()
	sessions := discovery.ClassifySessions(procs, cfg.CPUThreshold)
	sessions = discovery.FilterSessions(sessions, cfg.FilterProjects)
	discovery.SortSessions(sessions, cfg.SortColumn, cfg.SortAsc)
	return sessions
}

func main() {
	ensurePath()

	var cfg config.Config
	var intervalSec int
	var useCLI bool
	var sortCol string
	var sortDesc bool
	var filterPaths stringSliceFlag

	defaultInterval := 2
	if envVal := os.Getenv("CLAUDE_SESSIONS_INTERVAL"); envVal != "" {
		if n, err := strconv.Atoi(envVal); err == nil && n > 0 {
			defaultInterval = n
		}
	}

	flag.BoolVar(&cfg.Once, "once", false, "Print once and exit (terminal)")
	flag.BoolVar(&useCLI, "cli", false, "Use terminal dashboard instead of GUI")
	flag.BoolVar(&cfg.Compact, "compact", false, "Compact mode: show only status and project")
	flag.IntVar(&intervalSec, "interval", defaultInterval, "Refresh interval in seconds")
	flag.Float64Var(&cfg.CPUThreshold, "cpu-threshold", 3.0, "CPU% above which a session is considered working")
	flag.StringVar(&sortCol, "sort", "", "Sort by column: pid, status, uptime, cpu, mem, project, source, agents")
	flag.BoolVar(&sortDesc, "sort-desc", false, "Sort in descending order (default is ascending)")
	flag.Var(&filterPaths, "filter", "Only show sessions for this project path (can be repeated)")
	flag.Parse()

	cfg.Interval = time.Duration(intervalSec) * time.Second
	cfg.GUI = !useCLI && !cfg.Once

	cfg.FilterProjects = filterPaths

	if sortCol != "" {
		if !discovery.IsValidSortColumn(sortCol) {
			fmt.Fprintf(os.Stderr, "invalid sort column %q; valid columns: %v\n", sortCol, discovery.ValidSortColumns)
			os.Exit(1)
		}
		cfg.SortColumn = sortCol
		cfg.SortAsc = !sortDesc
	}

	// --once: print once to terminal and exit
	if cfg.Once {
		prevLines := 0
		sessions := fetchSessions(cfg)
		render.Draw(cfg, &prevLines, sessions, -1, "")
		return
	}

	// --gui (default): native Fyne window
	if cfg.GUI {
		gui.Run(cfg)
		return
	}

	// --cli: interactive terminal dashboard
	terminal.Init()

	if err := terminal.EnableRawMode(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: raw mode unavailable: %v\n", err)
	}

	cleanup := func() {
		terminal.DisableRawMode()
		fmt.Print(ansi.CursorShow)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		cleanup()
		os.Exit(0)
	}()

	keyCh := make(chan terminal.Key, 16)
	go terminal.ReadKeys(keyCh)

	fmt.Print(ansi.CursorHide + ansi.ClearScr)

	prevLines := 0
	selectedRow := 0
	focusErr := ""
	sessions := fetchSessions(cfg)

	for {
		// Clamp selection to valid range
		if len(sessions) == 0 {
			selectedRow = -1
		} else {
			if selectedRow < 0 {
				selectedRow = 0
			}
			if selectedRow >= len(sessions) {
				selectedRow = len(sessions) - 1
			}
		}

		render.Draw(cfg, &prevLines, sessions, selectedRow, focusErr)

		timer := time.NewTimer(cfg.Interval)
		resizeTick := time.NewTicker(50 * time.Millisecond)

	waitLoop:
		for {
			select {
			case <-timer.C:
				focusErr = "" // Clear stale error on refresh
				break waitLoop

			case key := <-keyCh:
				switch key {
				case terminal.KeyUp:
					if selectedRow > 0 {
						selectedRow--
					}
				case terminal.KeyDown:
					if selectedRow < len(sessions)-1 {
						selectedRow++
					}
				case terminal.KeyEnter:
					if selectedRow >= 0 && selectedRow < len(sessions) {
						if err := focus.FocusSession(sessions[selectedRow]); err != nil {
							focusErr = err.Error()
						} else {
							focusErr = ""
						}
					}
				case terminal.KeyQuit:
					cleanup()
					fmt.Println()
					return
				}
				// Redraw immediately on keypress
				timer.Stop()
				break waitLoop

			case <-resizeTick.C:
				if terminal.ResizeRequested.CompareAndSwap(1, 0) {
					fmt.Print(ansi.ClearScr)
					prevLines = 0
					timer.Stop()
					break waitLoop
				}
			}
		}
		resizeTick.Stop()

		// Refresh session data each iteration
		sessions = fetchSessions(cfg)
	}
}
