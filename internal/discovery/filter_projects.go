package discovery

import (
	"path/filepath"
	"strings"
)

// FilterSessions returns only sessions whose WorkDir matches at least one of the
// given project paths. Matching is done by comparing cleaned absolute paths.
// If filters is empty, all sessions are returned.
func FilterSessions(sessions []Session, filters []string) []Session {
	if len(filters) == 0 {
		return sessions
	}

	// Normalize filter paths: expand ~ and clean
	normalized := make([]string, 0, len(filters))
	for _, f := range filters {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if strings.HasPrefix(f, "~/") && HomeDir != "" {
			f = HomeDir + f[1:]
		}
		normalized = append(normalized, filepath.Clean(f))
	}
	if len(normalized) == 0 {
		return sessions
	}

	result := make([]Session, 0, len(sessions))
	for _, s := range sessions {
		wd := filepath.Clean(s.WorkDir)
		for _, f := range normalized {
			if wd == f {
				result = append(result, s)
				break
			}
		}
	}
	return result
}

// UniqueProjects returns the deduplicated list of WorkDir values from sessions.
func UniqueProjects(sessions []Session) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range sessions {
		if s.WorkDir != "" && !seen[s.WorkDir] {
			seen[s.WorkDir] = true
			result = append(result, s.WorkDir)
		}
	}
	return result
}
