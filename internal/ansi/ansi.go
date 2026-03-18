// Package ansi provides ANSI escape code constants and string helpers
// for measuring and manipulating strings that contain ANSI SGR sequences.
package ansi

import (
	"regexp"
	"strings"
)

// SGR escape codes for styling terminal output.
const (
	Bold       = "\033[1m"
	Dim        = "\033[2m"
	Reset      = "\033[0m"
	Yellow     = "\033[33m"
	Blue       = "\033[34m"
	Cyan       = "\033[36m"
	White      = "\033[37m"
	BgBlue     = "\033[44m"
	ClearEOL   = "\033[K"
	CursorHome = "\033[H"
	CursorHide = "\033[?25l"
	CursorShow = "\033[?25h"
	ClearScr   = "\033[2J"
	Reverse    = "\033[7m"
)

// Re matches ANSI SGR escape sequences for stripping/measuring visible text.
var Re = regexp.MustCompile(`\033\[[0-9;]*m`)

// VisibleLen returns the number of visible (non-ANSI) runes in s.
func VisibleLen(s string) int {
	return len([]rune(Re.ReplaceAllString(s, "")))
}

// Pad right-pads s with spaces to reach the given visible width.
func Pad(s string, width int) string {
	pad := width - VisibleLen(s)
	if pad <= 0 {
		return s
	}
	return s + strings.Repeat(" ", pad)
}

// Truncate truncates s to maxVisible visible characters, adding "…" if truncated.
// ANSI escape sequences are preserved up to the truncation point and properly terminated.
func Truncate(s string, maxVisible int) string {
	if maxVisible <= 0 {
		return ""
	}
	vis := 0
	inEsc := false
	lastNonEsc := 0
	for i, r := range s {
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		if r == '\033' {
			inEsc = true
			continue
		}
		vis++
		if vis > maxVisible {
			return s[:lastNonEsc] + Reset + "…"
		}
		lastNonEsc = i + len(string(r))
	}
	return s
}

// Clamp restricts v to the range [lo, hi].
func Clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
