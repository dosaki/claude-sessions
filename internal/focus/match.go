package focus

import (
	"path/filepath"
	"strings"

	"claude-sessions/internal/discovery"
)

// bestMatch selects the window that best matches the given session.
// It returns the best window and true, or an empty Window and false if
// no window matched at all.
//
// Matching priority (highest first):
//  1. Exact WorkDir match
//  2. Fuzzy title match — window title contains the project directory basename
//     (case-insensitive). If multiple windows match, the one containing more
//     path components of the WorkDir wins.
func bestMatch(windows []Window, s discovery.Session) (Window, bool) {
	if len(windows) == 0 || s.WorkDir == "" {
		return Window{}, false
	}

	// Phase 1: exact WorkDir match
	for _, w := range windows {
		if w.WorkDir != "" && w.WorkDir == s.WorkDir {
			return w, true
		}
	}

	// Phase 2: fuzzy title match on project basename
	basename := filepath.Base(s.WorkDir)
	if basename == "" || basename == "." || basename == "/" {
		return Window{}, false
	}

	baseLower := strings.ToLower(basename)
	workDirLower := strings.ToLower(s.WorkDir)

	type candidate struct {
		window Window
		score  int // higher = more path components matched in title
	}
	var best *candidate

	for _, w := range windows {
		if w.Title == "" {
			continue
		}
		titleLower := strings.ToLower(w.Title)
		if !strings.Contains(titleLower, baseLower) {
			continue
		}

		// Score: count how many path components of WorkDir appear in the title.
		// More components = more specific match.
		score := 1
		if strings.Contains(titleLower, workDirLower) {
			score = 100 // full path match is the strongest fuzzy signal
		} else {
			// Count additional path components present in the title
			parts := strings.Split(s.WorkDir, string(filepath.Separator))
			for _, part := range parts {
				if part != "" && strings.Contains(titleLower, strings.ToLower(part)) {
					score++
				}
			}
		}

		if best == nil || score > best.score {
			best = &candidate{window: w, score: score}
		}
	}

	if best != nil {
		return best.window, true
	}
	return Window{}, false
}
