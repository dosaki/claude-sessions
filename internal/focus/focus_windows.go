//go:build windows

package focus

import (
	"fmt"
	"os/exec"

	"claude-sessions/internal/discovery"
)

// FocusSession attempts to bring the IDE running the given session to the
// foreground. On Windows only IDE CLI commands are supported.
func FocusSession(s discovery.Session) error {
	switch s.Source {
	case "VSCode":
		return exec.Command("code", s.WorkDir).Start()
	case "Cursor":
		return exec.Command("cursor", s.WorkDir).Start()
	case "Windsurf":
		return exec.Command("windsurf", s.WorkDir).Start()
	default:
		return fmt.Errorf("focus not supported for %s on Windows", s.Source)
	}
}
