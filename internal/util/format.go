// Package util provides shared formatting helpers used by both the GUI and CLI renderers.
package util

import (
	"fmt"
	"path/filepath"
	"strings"

	"claude-sessions/internal/discovery"
)

// ShortenPath replaces the user's home directory prefix with "~".
func ShortenPath(path string) string {
	home := discovery.HomeDir
	if home != "" && strings.HasPrefix(path, home) && (len(path) == len(home) || path[len(home)] == '/') {
		path = "~" + path[len(home):]
	}
	return path
}

// FormatPercent formats a float percentage for display.
// Zero values are returned as "—" (em dash).
func FormatPercent(val float64) string {
	if val == 0 {
		return "—"
	}
	return fmt.Sprintf("%.1f%%", val)
}

// TruncateProject shortens a project path to its last two components
// when it exceeds maxLen visible characters.
func TruncateProject(project string, maxLen int) string {
	if len(project) > maxLen {
		sep := string(filepath.Separator)
		parts := strings.Split(project, sep)
		if len(parts) >= 2 {
			project = "…" + sep + strings.Join(parts[len(parts)-2:], sep)
		}
	}
	return project
}
