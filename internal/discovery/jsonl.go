package discovery

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Bytes to read from the tail of JSONL files.
const jsonlTailBytes = 16384

// HomeDir is the user's home directory, resolved once at startup.
var HomeDir string

func init() {
	// Best-effort: if the home directory can't be resolved, JSONL-based
	// status detection is silently disabled (HomeDir stays empty).
	HomeDir, _ = os.UserHomeDir()
}

// SessionState represents the parsed state from the last JSONL entry.
type SessionState struct {
	Type       string // "assistant", "user", "last-prompt", etc.
	StopReason string // "end_turn", "tool_use", etc. (only for assistant messages)
}

var resumeRe = regexp.MustCompile(`--resume\s+([0-9a-f-]{36})`)

// encodeWorkDir normalises a working directory path into the encoding
// Claude Code uses for its project directory names (slashes replaced by dashes).
func encodeWorkDir(workDir string) string {
	normalized := filepath.ToSlash(workDir)
	return strings.ReplaceAll(normalized, "/", "-")
}

// getSessionID extracts the session UUID from the command line (--resume flag)
// or finds the most recently modified .jsonl in the project directory.
func getSessionID(cmd, workDir string) string {
	if m := resumeRe.FindStringSubmatch(cmd); len(m) == 2 {
		return m[1]
	}

	if workDir == "" || HomeDir == "" {
		return ""
	}

	encoded := encodeWorkDir(workDir)
	projectDir := filepath.Join(HomeDir, ".claude", "projects", encoded)

	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return ""
	}

	var newest string
	var newestTime time.Time
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(newestTime) {
			newestTime = info.ModTime()
			newest = strings.TrimSuffix(e.Name(), ".jsonl")
		}
	}
	return newest
}

// getSessionFilePath returns the full path to a session's JSONL file.
func getSessionFilePath(sessionID, workDir string) string {
	if sessionID == "" || workDir == "" || HomeDir == "" {
		return ""
	}
	encoded := encodeWorkDir(workDir)
	return filepath.Join(HomeDir, ".claude", "projects", encoded, sessionID+".jsonl")
}

// readSessionState scans the tail of a JSONL file backward to find the last
// meaningful entry — one that reliably indicates whether the session is idle or
// waiting. It handles two JSONL styles:
//   - Consolidated: a single assistant entry with stop_reason ("end_turn", "tool_use")
//   - Streaming: per-block assistant entries with stop_reason null; the content
//     type of the last block indicates intent (tool_use → waiting, text → waiting)
func readSessionState(path string) SessionState {
	if path == "" {
		return SessionState{}
	}

	f, err := os.Open(path)
	if err != nil {
		return SessionState{}
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return SessionState{}
	}

	// Read only the tail — enough to contain the final JSON entry.
	offset := fi.Size() - jsonlTailBytes
	if offset < 0 {
		offset = 0
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return SessionState{}
	}

	buf, err := io.ReadAll(f)
	if err != nil {
		return SessionState{}
	}

	// Scan backward for the last meaningful entry. Track the most recent
	// streaming assistant entry's content type in case there's no consolidated
	// entry with a definitive stop_reason.
	lines := strings.Split(strings.TrimSpace(string(buf)), "\n")
	lastStreamingContentType := "" // "tool_use", "text", etc.
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		var entry struct {
			Type    string `json:"type"`
			Message struct {
				StopReason string          `json:"stop_reason"`
				Content    json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal([]byte(line), &entry) != nil || entry.Type == "" {
			continue
		}

		switch entry.Type {
		case "user":
			// If we saw streaming assistant entries after this user entry,
			// Claude has responded (in streaming style) and is waiting.
			if lastStreamingContentType != "" {
				return SessionState{
					Type:       "assistant",
					StopReason: inferStopReason(lastStreamingContentType),
				}
			}
			return SessionState{Type: "user"}
		case "assistant":
			// Definitive stop_reason → return immediately.
			if entry.Message.StopReason != "" {
				return SessionState{
					Type:       "assistant",
					StopReason: entry.Message.StopReason,
				}
			}
			// Streaming entry (stop_reason null): record content type if meaningful.
			if lastStreamingContentType == "" {
				lastStreamingContentType = lastContentType(entry.Message.Content)
			}
		default:
			// Skip progress, system, thinking, etc.
		}
	}

	// Reached start of buffer; if we saw streaming assistant entries, use them.
	if lastStreamingContentType != "" {
		return SessionState{
			Type:       "assistant",
			StopReason: inferStopReason(lastStreamingContentType),
		}
	}
	return SessionState{}
}

// lastContentType returns the type of the last content block in an assistant
// message's content array (e.g., "text", "tool_use", "thinking").
func lastContentType(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var blocks []struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(raw, &blocks) != nil || len(blocks) == 0 {
		return ""
	}
	return blocks[len(blocks)-1].Type
}

// inferStopReason maps a content block type to the equivalent stop_reason.
func inferStopReason(contentType string) string {
	if contentType == "tool_use" {
		return "tool_use"
	}
	return "end_turn"
}
