package discovery

import "testing"

func TestFilterSessions_NoFilter(t *testing.T) {
	sessions := []Session{{WorkDir: "/a"}, {WorkDir: "/b"}}
	result := FilterSessions(sessions, nil)
	if len(result) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(result))
	}
	result = FilterSessions(sessions, []string{})
	if len(result) != 2 {
		t.Errorf("expected 2 sessions with empty filter, got %d", len(result))
	}
}

func TestFilterSessions_MatchExact(t *testing.T) {
	sessions := []Session{
		{WorkDir: "/home/user/project-a"},
		{WorkDir: "/home/user/project-b"},
		{WorkDir: "/home/user/project-c"},
	}
	result := FilterSessions(sessions, []string{"/home/user/project-a", "/home/user/project-c"})
	if len(result) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(result))
	}
	if result[0].WorkDir != "/home/user/project-a" || result[1].WorkDir != "/home/user/project-c" {
		t.Errorf("unexpected sessions: %v", result)
	}
}

func TestFilterSessions_NoMatch(t *testing.T) {
	sessions := []Session{{WorkDir: "/a"}, {WorkDir: "/b"}}
	result := FilterSessions(sessions, []string{"/c"})
	if len(result) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(result))
	}
}

func TestFilterSessions_TildeExpansion(t *testing.T) {
	oldHome := HomeDir
	defer func() { HomeDir = oldHome }()
	HomeDir = "/home/testuser"

	sessions := []Session{{WorkDir: "/home/testuser/myproject"}}
	result := FilterSessions(sessions, []string{"~/myproject"})
	if len(result) != 1 {
		t.Errorf("expected 1 session with ~ expansion, got %d", len(result))
	}
}

func TestUniqueProjects(t *testing.T) {
	sessions := []Session{
		{WorkDir: "/a"},
		{WorkDir: "/b"},
		{WorkDir: "/a"},
		{WorkDir: ""},
		{WorkDir: "/c"},
	}
	result := UniqueProjects(sessions)
	if len(result) != 3 {
		t.Errorf("expected 3 unique projects, got %d: %v", len(result), result)
	}
}
