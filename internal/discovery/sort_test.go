package discovery

import "testing"

func TestSortSessionsByCPU(t *testing.T) {
	sessions := []Session{
		{PID: 1, CPU: 1.0},
		{PID: 2, CPU: 10.0},
		{PID: 3, CPU: 5.0},
	}

	SortSessions(sessions, "cpu", true)
	if sessions[0].PID != 1 || sessions[1].PID != 3 || sessions[2].PID != 2 {
		t.Errorf("ascending CPU sort failed: got PIDs %d,%d,%d", sessions[0].PID, sessions[1].PID, sessions[2].PID)
	}

	SortSessions(sessions, "cpu", false)
	if sessions[0].PID != 2 || sessions[1].PID != 3 || sessions[2].PID != 1 {
		t.Errorf("descending CPU sort failed: got PIDs %d,%d,%d", sessions[0].PID, sessions[1].PID, sessions[2].PID)
	}
}

func TestSortSessionsByStatus(t *testing.T) {
	sessions := []Session{
		{PID: 1, Status: "idle"},
		{PID: 2, Status: "working"},
		{PID: 3, Status: "waiting"},
	}

	SortSessions(sessions, "status", true)
	if sessions[0].Status != "working" || sessions[1].Status != "waiting" || sessions[2].Status != "idle" {
		t.Errorf("status sort failed: got %s,%s,%s", sessions[0].Status, sessions[1].Status, sessions[2].Status)
	}
}

func TestSortSessionsByUptime(t *testing.T) {
	sessions := []Session{
		{PID: 1, Elapsed: "01h30m"},
		{PID: 2, Elapsed: "10m05s"},
		{PID: 3, Elapsed: "01-00:00:00"},
	}

	SortSessions(sessions, "uptime", true)
	if sessions[0].PID != 2 || sessions[1].PID != 1 || sessions[2].PID != 3 {
		t.Errorf("uptime sort failed: got PIDs %d,%d,%d", sessions[0].PID, sessions[1].PID, sessions[2].PID)
	}
}

func TestParseElapsed(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"10m05s", 605},
		{"01h30m", 5400},
		{"01-00:00:00", 86400}, // 1 day
		{"2d03h", 2*86400 + 3*3600},
	}
	for _, tt := range tests {
		got := parseElapsed(tt.input)
		if got != tt.want {
			t.Errorf("parseElapsed(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestSortSessionsEmpty(t *testing.T) {
	SortSessions(nil, "cpu", true)
	SortSessions([]Session{}, "cpu", true)
	SortSessions([]Session{{PID: 1}}, "", true)
}

func TestIsValidSortColumn(t *testing.T) {
	if !IsValidSortColumn("cpu") {
		t.Error("cpu should be valid")
	}
	if IsValidSortColumn("invalid") {
		t.Error("invalid should not be valid")
	}
}
