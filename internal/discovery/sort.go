package discovery

import (
	"sort"
	"strconv"
	"strings"
)

// ValidSortColumns lists columns that sessions can be sorted by.
var ValidSortColumns = []string{"pid", "status", "uptime", "cpu", "mem", "project", "source", "agents"}

// IsValidSortColumn returns true if col is a recognized sort column name.
func IsValidSortColumn(col string) bool {
	for _, v := range ValidSortColumns {
		if v == col {
			return true
		}
	}
	return false
}

// parseElapsed converts an elapsed string to total seconds.
// Supports formats: "12m30s", "03h15m", "1d02h", "01-23:45:12" (dd-HH:MM:SS), "23:45:12" (HH:MM:SS).
func parseElapsed(s string) int {
	// Handle "DD-HH:MM:SS" or "HH:MM:SS" format (from ps)
	days := 0
	rest := s
	if idx := strings.Index(s, "-"); idx >= 0 {
		d, err := strconv.Atoi(s[:idx])
		if err == nil {
			days = d
		}
		rest = s[idx+1:]
	}
	if strings.Contains(rest, ":") {
		parts := strings.Split(rest, ":")
		total := days * 86400
		for i, p := range parts {
			n, _ := strconv.Atoi(p)
			switch len(parts) - i {
			case 3:
				total += n * 3600
			case 2:
				total += n * 60
			case 1:
				total += n
			}
		}
		return total
	}

	// Handle "1d02h", "03h15m", "12m30s" format
	total := days * 86400
	num := ""
	for _, c := range rest {
		if c >= '0' && c <= '9' {
			num += string(c)
		} else {
			n, _ := strconv.Atoi(num)
			num = ""
			switch c {
			case 'd':
				total += n * 86400
			case 'h':
				total += n * 3600
			case 'm':
				total += n * 60
			case 's':
				total += n
			}
		}
	}
	return total
}

// statusRank maps status strings to a sort rank so working > waiting > idle.
func statusRank(s string) int {
	switch s {
	case "working":
		return 0
	case "waiting":
		return 1
	default:
		return 2
	}
}

// SortSessions sorts sessions in place by the given column. If asc is true, sorts ascending.
func SortSessions(sessions []Session, column string, asc bool) {
	if len(sessions) < 2 || column == "" {
		return
	}

	sort.SliceStable(sessions, func(i, j int) bool {
		var less bool
		switch column {
		case "pid":
			less = sessions[i].PID < sessions[j].PID
		case "status":
			less = statusRank(sessions[i].Status) < statusRank(sessions[j].Status)
		case "uptime":
			less = parseElapsed(sessions[i].Elapsed) < parseElapsed(sessions[j].Elapsed)
		case "cpu":
			less = sessions[i].CPU < sessions[j].CPU
		case "mem":
			less = sessions[i].Mem < sessions[j].Mem
		case "project":
			less = strings.ToLower(sessions[i].WorkDir) < strings.ToLower(sessions[j].WorkDir)
		case "source":
			less = strings.ToLower(sessions[i].Source) < strings.ToLower(sessions[j].Source)
		case "agents":
			less = sessions[i].SubagentCount < sessions[j].SubagentCount
		default:
			return false
		}
		if !asc {
			return !less
		}
		return less
	})
}
