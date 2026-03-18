package discovery

import "testing"

func TestIsClaudeLine(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want bool
	}{
		// Positive matches
		{"bare_claude", "claude", true},
		{"path_claude", "/usr/local/bin/claude", true},
		{"claude_with_args", "claude --resume abc", true},
		{"path_claude_with_args", "/usr/local/bin/claude --help", true},
		{"backslash_claude", `C:\Users\bin\claude`, true},
		{"claude_exe", `C:\Users\bin\claude.exe`, true},
		{"unix_claude_exe", "/usr/bin/claude.exe", true},
		{"path_claude_space", "/usr/bin/claude --once", true},
		{"backslash_claude_space", `C:\bin\claude --flag`, true},

		// Negative: excluded patterns
		{"this_tool", "claude-sessions --once", false},
		{"desktop_app", "/Applications/Claude.app/Contents/MacOS/Claude", false},
		{"helper", "Claude Helper (GPU)", false},
		{"crashpad", "/path/to/crashpad_handler", false},
		{"shipit", "/path/to/ShipIt", false},

		// Negative: not claude
		{"unrelated", "vim main.go", false},
		{"partial_match", "claudette", false},
		{"empty", "", false},
		{"claude_substring", "my-claude-tool", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsClaudeLine(tt.cmd); got != tt.want {
				t.Errorf("IsClaudeLine(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestDetectIDEFromCmd(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{"vscode", "/home/user/.vscode/extensions/claude --resume abc", "VSCode"},
		{"code_server", "code-server --enable-proposed claude", "VSCode"},
		{"cursor", "/path/to/cursor/bin/claude --flag", "Cursor"},
		{"windsurf", "windsurf-extension/claude --resume xyz", "Windsurf"},
		{"case_insensitive", "/path/VSCode/bin/claude", "VSCode"},
		{"no_match", "/usr/local/bin/claude --help", ""},
		{"empty", "", ""},
		{"plain_cli", "claude", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectIDEFromCmd(tt.cmd); got != tt.want {
				t.Errorf("DetectIDEFromCmd(%q) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}
