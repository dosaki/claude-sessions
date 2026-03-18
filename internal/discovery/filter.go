package discovery

import "strings"

// excludePatterns are command substrings that indicate a process is NOT
// a Claude Code CLI session (e.g. the desktop app, helpers, this tool).
var excludePatterns = []string{
	"claude-sessions",
	"Claude.app",
	"Claude Helper",
	"crashpad_handler",
	"ShipIt",
}

// IsClaudeLine returns true if the command string looks like a Claude Code CLI process.
func IsClaudeLine(cmdPart string) bool {
	for _, pat := range excludePatterns {
		if strings.Contains(cmdPart, pat) {
			return false
		}
	}
	return strings.HasSuffix(cmdPart, "/claude") ||
		strings.HasSuffix(cmdPart, `\claude`) ||
		strings.HasSuffix(cmdPart, `\claude.exe`) ||
		strings.HasSuffix(cmdPart, "/claude.exe") ||
		strings.HasPrefix(cmdPart, "claude ") ||
		cmdPart == "claude" ||
		strings.Contains(cmdPart, "/claude ") ||
		strings.Contains(cmdPart, `\claude `)
}

// idePatterns maps command-line substrings to IDE names. Checked in order.
var idePatterns = []struct {
	substrings []string
	name       string
}{
	{[]string{"vscode", "code-server", ".vscode"}, "VSCode"},
	{[]string{"cursor"}, "Cursor"},
	{[]string{"windsurf"}, "Windsurf"},
}

// DetectIDEFromCmd checks the command string for known IDE markers.
// Returns the IDE name, or "" if no match.
func DetectIDEFromCmd(cmd string) string {
	lower := strings.ToLower(cmd)
	for _, p := range idePatterns {
		for _, sub := range p.substrings {
			if strings.Contains(lower, sub) {
				return p.name
			}
		}
	}
	return ""
}
