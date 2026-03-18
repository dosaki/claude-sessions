package discovery

import "testing"

func TestDetermineStatus(t *testing.T) {
	tests := []struct {
		name         string
		cpu          float64
		cpuThreshold float64
		state        SessionState
		want         string
	}{
		// CPU above threshold always wins
		{"high_cpu_no_state", 20.0, 8.0, SessionState{}, "working"},
		{"high_cpu_with_idle_state", 15.0, 8.0, SessionState{Type: "assistant", StopReason: "end_turn"}, "working"},
		{"high_cpu_with_tool_use", 10.0, 8.0, SessionState{Type: "assistant", StopReason: "tool_use"}, "working"},

		// CPU below threshold — JSONL determines status
		{"low_cpu_end_turn", 1.0, 8.0, SessionState{Type: "assistant", StopReason: "end_turn"}, "idle"},
		{"low_cpu_tool_use", 1.0, 8.0, SessionState{Type: "assistant", StopReason: "tool_use"}, "waiting"},
		{"low_cpu_other_stop", 1.0, 8.0, SessionState{Type: "assistant", StopReason: "max_tokens"}, "idle"},
		{"low_cpu_no_stop_reason", 1.0, 8.0, SessionState{Type: "assistant"}, "idle"},

		// Non-assistant types default to idle
		{"user_message", 1.0, 8.0, SessionState{Type: "user"}, "idle"},
		{"empty_state", 0.0, 8.0, SessionState{}, "idle"},
		{"unknown_type", 2.0, 8.0, SessionState{Type: "system"}, "idle"},

		// Edge: CPU exactly at threshold (not above)
		{"at_threshold", 8.0, 8.0, SessionState{Type: "assistant", StopReason: "tool_use"}, "waiting"},

		// Edge: zero threshold
		{"zero_threshold_any_cpu", 0.1, 0.0, SessionState{}, "working"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineStatus(tt.cpu, tt.cpuThreshold, tt.state)
			if got != tt.want {
				t.Errorf("DetermineStatus(cpu=%.1f, thresh=%.1f, state=%+v) = %q, want %q",
					tt.cpu, tt.cpuThreshold, tt.state, got, tt.want)
			}
		})
	}
}
