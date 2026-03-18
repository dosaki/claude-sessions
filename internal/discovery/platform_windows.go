//go:build windows

package discovery

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type winProcEntry struct {
	PID        int     `json:"PID"`
	PPID       int     `json:"PPID"`
	Command    string  `json:"Command"`
	ElapsedSec int     `json:"ElapsedSec"`
	MemPct     float64 `json:"MemPct"`
}

// GetClaudeProcesses discovers running Claude Code processes via PowerShell.
func GetClaudeProcesses() []RawProc {
	script := `$totalMem = (Get-CimInstance Win32_ComputerSystem).TotalPhysicalMemory;` +
		`Get-CimInstance Win32_Process |
		Where-Object { $_.CommandLine -and $_.CommandLine -match 'claude' } |
		ForEach-Object {
			$elapsed = 0
			if ($_.CreationDate) { $elapsed = [int]((Get-Date) - $_.CreationDate).TotalSeconds }
			$memPct = 0
			if ($totalMem -gt 0 -and $_.WorkingSetSize) { $memPct = [math]::Round($_.WorkingSetSize / $totalMem * 100, 1) }
			[PSCustomObject]@{
				PID        = [int]$_.ProcessId
				PPID       = [int]$_.ParentProcessId
				Command    = $_.CommandLine
				ElapsedSec = $elapsed
				MemPct     = $memPct
			}
		} | ConvertTo-Json -Compress`

	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return nil
	}

	// PowerShell emits a bare object (not array) when there is exactly one result.
	raw := strings.TrimSpace(string(out))
	if raw == "" {
		return nil
	}

	var entries []winProcEntry
	if raw[0] == '[' {
		_ = json.Unmarshal([]byte(raw), &entries)
	} else {
		var single winProcEntry
		if json.Unmarshal([]byte(raw), &single) == nil {
			entries = append(entries, single)
		}
	}

	var procs []RawProc
	for _, e := range entries {
		if !IsClaudeLine(e.Command) {
			continue
		}
		procs = append(procs, RawProc{
			PID:     e.PID,
			PPID:    e.PPID,
			CPU:     0, // CPU% not easily available in a single snapshot on Windows
			Mem:     e.MemPct,
			Elapsed: formatElapsed(e.ElapsedSec),
			Command: e.Command,
		})
	}
	return procs
}

// formatElapsed converts seconds into a human-readable string matching
// the style used by Unix ps (e.g. "01:23", "02:15:30", "1-04:20:00").
func formatElapsed(totalSec int) string {
	if totalSec < 0 {
		totalSec = 0
	}
	days := totalSec / 86400
	remaining := totalSec % 86400
	hours := remaining / 3600
	remaining %= 3600
	mins := remaining / 60
	secs := remaining % 60

	if days > 0 {
		return fmt.Sprintf("%d-%02d:%02d:%02d", days, hours, mins, secs)
	}
	if hours > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", hours, mins, secs)
	}
	return fmt.Sprintf("%02d:%02d", mins, secs)
}

// GetWorkDir on Windows: retrieving the CWD of another process requires
// NtQueryInformationProcess, which is complex and may need elevation.
// We try a best-effort approach via PowerShell.
func GetWorkDir(pid int) string {
	script := fmt.Sprintf(
		`$p = Get-CimInstance Win32_Process -Filter "ProcessId=%d"; `+
			`if ($p.ExecutablePath) { Split-Path $p.ExecutablePath } else { "" }`,
		pid,
	)
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return ""
	}
	result := strings.TrimSpace(string(out))
	if result == "" {
		return ""
	}
	return result
}

// GetSource detects the IDE or terminal that launched a Claude session.
func GetSource(ppid int, cmd string) string {
	if ide := DetectIDEFromCmd(cmd); ide != "" {
		return ide
	}

	script := fmt.Sprintf(`(Get-Process -Id %d -ErrorAction SilentlyContinue).ProcessName`, ppid)
	out, err := exec.Command("powershell", "-NoProfile", "-Command", script).Output()
	if err != nil {
		return "CLI"
	}
	parent := strings.ToLower(strings.TrimSpace(string(out)))

	switch {
	case strings.Contains(parent, "code"):
		return "VSCode"
	case strings.Contains(parent, "cursor"):
		return "Cursor"
	case strings.Contains(parent, "windowsterminal"):
		return "Win Terminal"
	case strings.Contains(parent, "powershell") || strings.Contains(parent, "pwsh"):
		return "PowerShell"
	case strings.Contains(parent, "cmd"):
		return "CMD"
	case strings.Contains(parent, "alacritty"):
		return "Alacritty"
	case strings.Contains(parent, "wezterm"):
		return "WezTerm"
	}
	return "CLI"
}
