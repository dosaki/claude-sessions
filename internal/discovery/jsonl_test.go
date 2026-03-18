package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEncodeWorkDir(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"unix_path", "/home/user/projects/my-app", "-home-user-projects-my-app"},
		{"root", "/", "-"},
		{"empty", "", ""},
		{"forward_slash_windows_style", "C:/Users/dev/app", "C:-Users-dev-app"},
		{"trailing_slash", "/home/user/", "-home-user-"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := encodeWorkDir(tt.in); got != tt.want {
				t.Errorf("encodeWorkDir(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestGetSessionID_ResumeFlag(t *testing.T) {
	uuid := "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{"with_resume", "claude --resume " + uuid, uuid},
		{"resume_in_middle", "/usr/bin/claude --resume " + uuid + " --other", uuid},
		{"no_resume", "claude --help", ""},
		{"empty", "", ""},
		{"bad_uuid", "claude --resume not-a-uuid", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pass empty workDir so it only tests the regex path
			if got := getSessionID(tt.cmd, ""); got != tt.want {
				t.Errorf("getSessionID(%q, \"\") = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestGetSessionFilePath(t *testing.T) {
	origHome := HomeDir
	defer func() { HomeDir = origHome }()
	HomeDir = "/home/testuser"

	tests := []struct {
		name      string
		sessionID string
		workDir   string
		want      string
	}{
		{
			"normal",
			"abc-123",
			"/home/testuser/dev/myapp",
			filepath.Join("/home/testuser", ".claude", "projects", "-home-testuser-dev-myapp", "abc-123.jsonl"),
		},
		{"empty_session", "", "/home/user/dev", ""},
		{"empty_workdir", "abc-123", "", ""},
		{"both_empty", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getSessionFilePath(tt.sessionID, tt.workDir); got != tt.want {
				t.Errorf("getSessionFilePath(%q, %q) = %q, want %q", tt.sessionID, tt.workDir, got, tt.want)
			}
		})
	}
}

func TestGetSessionFilePath_NoHome(t *testing.T) {
	origHome := HomeDir
	defer func() { HomeDir = origHome }()
	HomeDir = ""

	if got := getSessionFilePath("abc", "/some/dir"); got != "" {
		t.Errorf("expected empty path when HomeDir is empty, got %q", got)
	}
}

func TestReadSessionState(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    SessionState
	}{
		{
			"assistant_end_turn",
			`{"type":"user","message":{"content":"hello"}}
{"type":"assistant","message":{"stop_reason":"end_turn","content":"hi"}}
`,
			SessionState{Type: "assistant", StopReason: "end_turn"},
		},
		{
			"assistant_tool_use",
			`{"type":"assistant","message":{"stop_reason":"tool_use"}}
`,
			SessionState{Type: "assistant", StopReason: "tool_use"},
		},
		{
			"user_last",
			`{"type":"assistant","message":{"stop_reason":"end_turn"}}
{"type":"user","message":{"content":"do something"}}
`,
			SessionState{Type: "user"},
		},
		{
			"empty_lines_ignored",
			`{"type":"assistant","message":{"stop_reason":"end_turn"}}

`,
			SessionState{Type: "assistant", StopReason: "end_turn"},
		},
		{
			"invalid_json",
			`not json at all`,
			SessionState{},
		},
		{
			"empty_file",
			"",
			SessionState{},
		},
		{
			"mixed_valid_invalid",
			`not json
{"type":"assistant","message":{"stop_reason":"tool_use"}}
`,
			SessionState{Type: "assistant", StopReason: "tool_use"},
		},
		{
			"streaming_then_definitive",
			`{"type":"assistant","message":{"stop_reason":"end_turn","content":"done"}}
{"type":"assistant","message":{"stop_reason":"","content":"streaming..."}}
`,
			SessionState{Type: "assistant", StopReason: "end_turn"},
		},
		{
			"progress_entries_skipped",
			`{"type":"user","message":{"content":"hello"}}
{"type":"progress","message":{"content":"running tool..."}}
`,
			SessionState{Type: "user"},
		},
		{
			"null_stop_reason_string_content_skipped_to_user",
			`{"type":"user","message":{"content":"tool result"}}
{"type":"assistant","message":{"stop_reason":null,"content":"partial"}}
`,
			SessionState{Type: "user"},
		},
		{
			"all_streaming_string_content_no_definitive",
			`{"type":"assistant","message":{"stop_reason":"","content":"chunk1"}}
{"type":"assistant","message":{"stop_reason":"","content":"chunk2"}}
`,
			SessionState{},
		},
		{
			"streaming_tool_use_after_user",
			`{"type":"user","message":{"content":"tool result"}}
{"type":"assistant","message":{"stop_reason":null,"content":[{"type":"thinking","thinking":"hmm"}]}}
{"type":"assistant","message":{"stop_reason":null,"content":[{"type":"text","text":"I will do that"}]}}
{"type":"assistant","message":{"stop_reason":null,"content":[{"type":"tool_use","id":"t1","name":"Bash","input":{}}]}}
{"type":"progress","message":{"content":"running"}}
`,
			SessionState{Type: "assistant", StopReason: "tool_use"},
		},
		{
			"streaming_text_after_user",
			`{"type":"user","message":{"content":"hello"}}
{"type":"assistant","message":{"stop_reason":null,"content":[{"type":"thinking","thinking":"hmm"}]}}
{"type":"assistant","message":{"stop_reason":null,"content":[{"type":"text","text":"Here is the answer"}]}}
{"type":"progress","message":{"content":"done"}}
`,
			SessionState{Type: "assistant", StopReason: "end_turn"},
		},
		{
			"streaming_only_thinking_after_user",
			`{"type":"user","message":{"content":"hello"}}
{"type":"assistant","message":{"stop_reason":null,"content":[{"type":"thinking","thinking":"hmm"}]}}
`,
			SessionState{Type: "assistant", StopReason: "end_turn"},
		},
		{
			"streaming_no_user_before",
			`{"type":"assistant","message":{"stop_reason":null,"content":[{"type":"text","text":"response"}]}}
{"type":"progress","message":{"content":"done"}}
`,
			SessionState{Type: "assistant", StopReason: "end_turn"},
		},
		{
			"thinking_and_system_skipped",
			`{"type":"assistant","message":{"stop_reason":"tool_use"}}
{"type":"thinking","message":{"content":"hmm"}}
{"type":"system","message":{"content":"init"}}
`,
			SessionState{Type: "assistant", StopReason: "tool_use"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := os.CreateTemp(t.TempDir(), "session-*.jsonl")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := f.WriteString(tt.content); err != nil {
				t.Fatal(err)
			}
			f.Close()

			got := readSessionState(f.Name())
			if got != tt.want {
				t.Errorf("readSessionState() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestReadSessionState_EmptyPath(t *testing.T) {
	got := readSessionState("")
	if got != (SessionState{}) {
		t.Errorf("readSessionState(\"\") = %+v, want empty", got)
	}
}

func TestReadSessionState_MissingFile(t *testing.T) {
	got := readSessionState("/nonexistent/path/session.jsonl")
	if got != (SessionState{}) {
		t.Errorf("readSessionState(missing) = %+v, want empty", got)
	}
}
