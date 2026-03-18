// Package config holds the runtime configuration for claude-sessions.
package config

import "time"

// Config holds the runtime configuration parsed from flags.
type Config struct {
	Once           bool
	GUI            bool
	Compact        bool
	Interval       time.Duration
	CPUThreshold   float64
	SortColumn     string   // column name to sort by (pid, status, uptime, cpu, mem, project, source, agents)
	SortAsc        bool     // true = ascending, false = descending
	FilterProjects []string // if non-empty, only show sessions whose WorkDir matches one of these paths
}
