// Package gui provides a native Fyne-based GUI dashboard for Claude sessions.
package gui

import (
	"fmt"
	"image/color"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"claude-sessions/internal/config"
	"claude-sessions/internal/discovery"
	"claude-sessions/internal/focus"
)

// Status colors matching the terminal dashboard.
var (
	colorWorking = color.NRGBA{R: 68, G: 138, B: 255, A: 255}
	colorWaiting = color.NRGBA{R: 255, G: 193, B: 7, A: 255}
	colorIdle    = color.NRGBA{R: 158, G: 158, B: 158, A: 255}
)

// Column indices for full mode.
const (
	colPID = iota
	colStatus
	colUptime
	colCPU
	colMem
	colProject
	colSource
	colAgents
	colCount
)

// Column indices for compact mode.
const (
	compactColStatus = iota
	compactColProject
	compactColCount
)

var columnHeaders = [colCount]string{"PID", "STATUS", "UPTIME", "CPU%", "MEM%", "PROJECT", "SOURCE", "AGENTS"}
var compactColumnHeaders = [compactColCount]string{"STATUS", "PROJECT"}

// columnSortKeys maps column indices to sort column name strings.
var columnSortKeys = [colCount]string{"pid", "status", "uptime", "cpu", "mem", "project", "source", "agents"}
var compactColumnSortKeys = [compactColCount]string{"status", "project"}

// dashboard holds the GUI state and provides thread-safe access to session data.
type dashboard struct {
	mu             sync.RWMutex
	sessions       []discovery.Session // filtered + sorted sessions for display
	allSessions    []discovery.Session // all sessions before filtering (for project discovery)
	working        int
	waiting        int
	idle           int
	compact        bool
	sortColumn     string
	sortAsc        bool
	filterProjects []string // if non-empty, only show these projects
	notifyWaiting  bool     // if true, play a sound when a session enters "waiting"
	prevWaiting    map[int]bool // PIDs that were already waiting last refresh
}

func (d *dashboard) refresh(cpuThreshold float64) {
	procs := discovery.GetClaudeProcesses()
	allSessions := discovery.ClassifySessions(procs, cpuThreshold)

	d.mu.Lock()
	d.allSessions = allSessions
	filtered := discovery.FilterSessions(allSessions, d.filterProjects)

	working, waiting, idle := 0, 0, 0
	nowWaiting := make(map[int]bool)
	newWaiting := false
	for _, s := range filtered {
		switch s.Status {
		case "working":
			working++
		case "waiting":
			waiting++
			nowWaiting[s.PID] = true
			if d.notifyWaiting && !d.prevWaiting[s.PID] {
				newWaiting = true
			}
		default:
			idle++
		}
	}
	d.prevWaiting = nowWaiting

	d.sessions = filtered
	d.working = working
	d.waiting = waiting
	d.idle = idle
	discovery.SortSessions(d.sessions, d.sortColumn, d.sortAsc)
	d.mu.Unlock()

	if newWaiting {
		go playNotificationSound()
	}
}

// allProjects returns the unique project paths from all (unfiltered) sessions.
func (d *dashboard) allProjects() []string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return discovery.UniqueProjects(d.allSessions)
}

// setFilter updates the filter and re-applies it to the current session data.
func (d *dashboard) setFilter(projects []string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.filterProjects = projects
	filtered := discovery.FilterSessions(d.allSessions, d.filterProjects)

	working, waiting, idle := 0, 0, 0
	for _, s := range filtered {
		switch s.Status {
		case "working":
			working++
		case "waiting":
			waiting++
		default:
			idle++
		}
	}
	d.sessions = filtered
	d.working = working
	d.waiting = waiting
	d.idle = idle
	discovery.SortSessions(d.sessions, d.sortColumn, d.sortAsc)
}

func (d *dashboard) toggleSort(col string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.sortColumn == col {
		d.sortAsc = !d.sortAsc
	} else {
		d.sortColumn = col
		d.sortAsc = true
	}
	discovery.SortSessions(d.sessions, d.sortColumn, d.sortAsc)
}

func (d *dashboard) headerText(name, sortKey string) string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.sortColumn == sortKey {
		if d.sortAsc {
			return name + " ▲"
		}
		return name + " ▼"
	}
	return name
}

func (d *dashboard) len() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.sessions)
}

func (d *dashboard) get(row int) discovery.Session {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if row < len(d.sessions) {
		return d.sessions[row]
	}
	return discovery.Session{}
}

func (d *dashboard) cols() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.compact {
		return compactColCount
	}
	return colCount
}

func (d *dashboard) isCompact() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.compact
}

func (d *dashboard) setCompact(v bool) {
	d.mu.Lock()
	d.compact = v
	d.mu.Unlock()
}

func (d *dashboard) summaryText() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	total := len(d.sessions)
	if total == 0 {
		return "No active Claude sessions"
	}
	parts := []string{fmt.Sprintf("%d session(s)", total)}
	if d.working > 0 {
		parts = append(parts, fmt.Sprintf("%d working", d.working))
	}
	if d.waiting > 0 {
		parts = append(parts, fmt.Sprintf("%d waiting", d.waiting))
	}
	if d.idle > 0 {
		parts = append(parts, fmt.Sprintf("%d idle", d.idle))
	}
	return strings.Join(parts, "  ·  ")
}

func shortenPath(path string) string {
	home := discovery.HomeDir
	if home != "" && strings.HasPrefix(path, home) && (len(path) == len(home) || path[len(home)] == '/') {
		path = "~" + path[len(home):]
	}
	return path
}

func statusColor(status string) color.NRGBA {
	switch status {
	case "working":
		return colorWorking
	case "waiting":
		return colorWaiting
	default:
		return colorIdle
	}
}

func statusLabel(status string) string {
	switch status {
	case "working":
		return "Working"
	case "waiting":
		return "Waiting for input"
	default:
		return "Idle"
	}
}

func formatAgentsGUI(count int) string {
	if count > 0 {
		return strconv.Itoa(count) + " running"
	}
	return "none"
}

func formatProjectGUI(workDir string) string {
	p := shortenPath(workDir)
	if p == "" {
		return "unknown"
	}
	// Shorten long paths to last 2 components
	if len(p) > 40 {
		sep := string(filepath.Separator)
		parts := strings.Split(p, sep)
		if len(parts) >= 2 {
			p = "…" + sep + strings.Join(parts[len(parts)-2:], sep)
		}
	}
	return p
}

func formatPercentGUI(val float64) string {
	if val == 0 {
		return "—"
	}
	return fmt.Sprintf("%.1f%%", val)
}

// playNotificationSound plays a short system alert sound.
func playNotificationSound() {
	switch runtime.GOOS {
	case "darwin":
		// macOS system glass sound — short and unobtrusive.
		exec.Command("afplay", "/System/Library/Sounds/Glass.aiff").Run()
	case "linux":
		// Try PulseAudio bell, fall back to BEL character.
		if err := exec.Command("paplay", "/usr/share/sounds/freedesktop/stereo/bell.oga").Run(); err != nil {
			fmt.Print("\a")
		}
	default:
		fmt.Print("\a") // terminal BEL
	}
}

// Run starts the Fyne GUI dashboard.
func Run(cfg config.Config) {
	a := app.New()
	a.Settings().SetTheme(&dashboardTheme{})

	w := a.NewWindow("Claude Sessions Dashboard")
	w.Resize(fyne.NewSize(1040, 500))

	d := &dashboard{
		compact:        cfg.Compact,
		sortColumn:     cfg.SortColumn,
		sortAsc:        cfg.SortAsc,
		filterProjects: cfg.FilterProjects,
		prevWaiting:    make(map[int]bool),
	}
	d.refresh(cfg.CPUThreshold)

	// Summary bar
	summaryLabel := widget.NewLabel(d.summaryText())
	summaryLabel.TextStyle = fyne.TextStyle{Bold: true}

	timeLabel := widget.NewLabel(time.Now().Format("2006-01-02 15:04:05"))

	var table *widget.Table // forward declaration for closures

	headerBar := container.NewHBox(
		summaryLabel,
		layout.NewSpacer(),
		timeLabel,
	)

	// Empty state
	emptyLabel := widget.NewLabel("No active Claude sessions found.\nStart a session with: claude")
	emptyLabel.Alignment = fyne.TextAlignCenter

	// setTableWidths configures column widths based on current mode.
	setTableWidths := func() {
		if d.isCompact() {
			table.SetColumnWidth(compactColStatus, 160)
			table.SetColumnWidth(compactColProject, 500)
		} else {
			table.SetColumnWidth(colPID, 70)
			table.SetColumnWidth(colStatus, 160)
			table.SetColumnWidth(colUptime, 100)
			table.SetColumnWidth(colCPU, 70)
			table.SetColumnWidth(colMem, 70)
			table.SetColumnWidth(colProject, 260)
			table.SetColumnWidth(colSource, 100)
			table.SetColumnWidth(colAgents, 100)
		}
	}

	// Table
	table = widget.NewTable(
		func() (int, int) {
			return d.len(), d.cols()
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("template")
			dot := canvas.NewCircle(colorIdle)
			dot.Resize(fyne.NewSize(10, 10))
			statusText := canvas.NewText("Status", colorIdle)
			statusText.TextSize = 13
			statusBox := container.NewHBox(dot, statusText)
			return container.NewStack(label, statusBox)
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			c := cell.(*fyne.Container)
			label := c.Objects[0].(*widget.Label)
			statusBox := c.Objects[1].(*fyne.Container)

			s := d.get(id.Row)
			compact := d.isCompact()

			// Determine which logical column this is
			isStatusCol := (compact && id.Col == compactColStatus) || (!compact && id.Col == colStatus)
			isProjectCol := (compact && id.Col == compactColProject) || (!compact && id.Col == colProject)

			if isStatusCol {
				label.Hide()
				statusBox.Show()
				dot := statusBox.Objects[0].(*canvas.Circle)
				text := statusBox.Objects[1].(*canvas.Text)
				clr := statusColor(s.Status)
				dot.FillColor = clr
				dot.Refresh()
				text.Text = statusLabel(s.Status)
				text.Color = clr
				text.Refresh()
			} else {
				statusBox.Hide()
				label.Show()
				if isProjectCol {
					label.SetText(formatProjectGUI(s.WorkDir))
				} else if !compact {
					switch id.Col {
					case colPID:
						label.SetText(strconv.Itoa(s.PID))
					case colUptime:
						label.SetText(s.Elapsed)
					case colCPU:
						label.SetText(formatPercentGUI(s.CPU))
					case colMem:
						label.SetText(formatPercentGUI(s.Mem))
					case colSource:
						label.SetText(s.Source)
					case colAgents:
						label.SetText(formatAgentsGUI(s.SubagentCount))
					}
				}
			}
		},
	)

	setTableWidths()

	// Click a row to focus/activate the session's terminal or IDE.
	table.OnSelected = func(id widget.TableCellID) {
		s := d.get(id.Row)
		if s.PID != 0 {
			go func() {
				if err := focus.FocusSession(s); err != nil {
					fyne.Do(func() {
						dialog.ShowError(fmt.Errorf("Could not focus session (PID %d, %s):\n%s", s.PID, s.Source, err), w)
					})
				}
			}()
		}
		table.UnselectAll()
	}

	// Custom clickable column header row.
	// Each header is a flat button that toggles sort on click.
	var headerBtns [colCount]*widget.Button
	var compactHeaderBtns [compactColCount]*widget.Button

	refreshHeaders := func() {
		if d.isCompact() {
			for i := range compactHeaderBtns {
				compactHeaderBtns[i].SetText(d.headerText(compactColumnHeaders[i], compactColumnSortKeys[i]))
			}
		} else {
			for i := range headerBtns {
				headerBtns[i].SetText(d.headerText(columnHeaders[i], columnSortKeys[i]))
			}
		}
	}

	// Full-mode header buttons
	for i := range headerBtns {
		sortKey := columnSortKeys[i]
		headerBtns[i] = widget.NewButton(d.headerText(columnHeaders[i], sortKey), func() {
			d.toggleSort(sortKey)
			refreshHeaders()
			table.Refresh()
		})
		headerBtns[i].Importance = widget.LowImportance
	}

	// Compact-mode header buttons
	for i := range compactHeaderBtns {
		sortKey := compactColumnSortKeys[i]
		compactHeaderBtns[i] = widget.NewButton(d.headerText(compactColumnHeaders[i], sortKey), func() {
			d.toggleSort(sortKey)
			refreshHeaders()
			table.Refresh()
		})
		compactHeaderBtns[i].Importance = widget.LowImportance
	}

	// Column widths for full mode — must match setTableWidths.
	fullWidths := [colCount]float32{70, 160, 100, 70, 70, 260, 100, 100}
	compactWidths := [compactColCount]float32{160, 500}

	var columnHeaderRow *fyne.Container

	buildHeaderRow := func() *fyne.Container {
		if d.isCompact() {
			objs := make([]fyne.CanvasObject, compactColCount)
			for i := range compactHeaderBtns {
				objs[i] = container.New(layout.NewMaxLayout(), container.NewGridWrap(fyne.NewSize(compactWidths[i], 30), compactHeaderBtns[i]))
			}
			return container.NewHBox(objs...)
		}
		objs := make([]fyne.CanvasObject, colCount)
		for i := range headerBtns {
			objs[i] = container.New(layout.NewMaxLayout(), container.NewGridWrap(fyne.NewSize(fullWidths[i], 30), headerBtns[i]))
		}
		return container.NewHBox(objs...)
	}

	columnHeaderRow = buildHeaderRow()

	// Forward-declare updateVisibility so filter callback can use it.
	var updateVisibility func()

	// tableWithHeaders wraps the column header row and the table in a border layout.
	tableWithHeaders := container.NewBorder(columnHeaderRow, nil, nil, nil, table)

	rebuildColumnHeaders := func() {
		newRow := buildHeaderRow()
		columnHeaderRow.Objects = newRow.Objects
		columnHeaderRow.Layout = newRow.Layout
		columnHeaderRow.Refresh()
	}

	// Wire compact toggle now that table exists.
	compactToggle := widget.NewCheck("Compact", func(checked bool) {
		d.setCompact(checked)
		setTableWidths()
		rebuildColumnHeaders()
		refreshHeaders()
		table.Refresh()
	})
	compactToggle.Checked = cfg.Compact

	// Notification toggle — play a sound when a session starts waiting for input.
	notifyToggle := widget.NewCheck("Notify", func(checked bool) {
		d.mu.Lock()
		d.notifyWaiting = checked
		if checked {
			// Seed current waiting set so existing sessions don't trigger immediately.
			d.prevWaiting = make(map[int]bool)
			for _, s := range d.sessions {
				if s.Status == "waiting" {
					d.prevWaiting[s.PID] = true
				}
			}
		}
		d.mu.Unlock()
	})

	// Filter button — opens a dialog with checkboxes for each discovered project.
	filterBtn := widget.NewButton("Filter", nil)
	filterBtn.OnTapped = func() {
		projects := d.allProjects()
		if len(projects) == 0 {
			dialog.ShowInformation("Filter Projects", "No projects discovered yet.", w)
			return
		}

		// Build a set of currently active filters for pre-checking
		d.mu.RLock()
		activeFilters := make(map[string]bool, len(d.filterProjects))
		for _, f := range d.filterProjects {
			activeFilters[f] = true
		}
		noFilter := len(d.filterProjects) == 0
		d.mu.RUnlock()

		checks := make([]*widget.Check, len(projects))
		for i, p := range projects {
			checked := noFilter || activeFilters[p]
			checks[i] = widget.NewCheck(shortenPath(p), nil)
			checks[i].Checked = checked
		}

		// "Select All" / "Clear All" buttons
		selectAll := widget.NewButton("Select All", func() {
			for _, c := range checks {
				c.SetChecked(true)
			}
		})
		clearAll := widget.NewButton("Clear All", func() {
			for _, c := range checks {
				c.SetChecked(false)
			}
		})
		btnRow := container.NewHBox(selectAll, clearAll)

		// Build the list of checkboxes
		items := make([]fyne.CanvasObject, 0, len(checks)+1)
		items = append(items, btnRow)
		for _, c := range checks {
			items = append(items, c)
		}
		content := container.NewVBox(items...)
		scroll := container.NewVScroll(content)
		scroll.SetMinSize(fyne.NewSize(400, 300))

		dialog.ShowCustomConfirm("Filter Projects", "Apply", "Cancel", scroll, func(ok bool) {
			if !ok {
				return
			}
			var selected []string
			allChecked := true
			for i, c := range checks {
				if c.Checked {
					selected = append(selected, projects[i])
				} else {
					allChecked = false
				}
			}
			// If all are checked, treat as "no filter"
			if allChecked {
				selected = nil
			}
			d.setFilter(selected)
			summaryLabel.SetText(d.summaryText())
			table.Refresh()
			updateVisibility()
		}, w)
	}

	// Insert controls before timeLabel (last element) in the header bar.
	objs := headerBar.Objects
	headerBar.Objects = append(objs[:len(objs)-1], compactToggle, notifyToggle, filterBtn, objs[len(objs)-1])
	headerBar.Refresh()

	// Layout: show table or empty state
	content := container.NewStack(tableWithHeaders, container.NewCenter(emptyLabel))

	updateVisibility = func() {
		if d.len() > 0 {
			tableWithHeaders.Show()
			emptyLabel.Hide()
		} else {
			tableWithHeaders.Hide()
			emptyLabel.Show()
		}
	}
	updateVisibility()

	w.SetContent(container.NewBorder(headerBar, nil, nil, nil, content))

	// Refresh loop — data fetch runs in background, UI updates via fyne.Do.
	go func() {
		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()
		for range ticker.C {
			d.refresh(cfg.CPUThreshold)
			fyne.Do(func() {
				summaryLabel.SetText(d.summaryText())
				timeLabel.SetText(time.Now().Format("2006-01-02 15:04:05"))
				table.Refresh()
				updateVisibility()
			})
		}
	}()

	w.ShowAndRun()
}
