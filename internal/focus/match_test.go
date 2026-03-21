package focus

import (
	"testing"

	"claude-sessions/internal/discovery"
)

func TestBestMatch_ExactWorkDir(t *testing.T) {
	windows := []Window{
		{AppName: "Ghostty", Title: "zsh — ~/other", WorkDir: "/Users/x/other", Index: 1},
		{AppName: "Ghostty", Title: "zsh — ~/myproject", WorkDir: "/Users/x/myproject", Index: 2},
		{AppName: "Ghostty", Title: "zsh — ~/third", WorkDir: "/Users/x/third", Index: 3},
	}
	s := discovery.Session{WorkDir: "/Users/x/myproject"}

	got, ok := bestMatch(windows, s)
	if !ok {
		t.Fatal("expected a match")
	}
	if got.Index != 2 {
		t.Errorf("expected window index 2, got %d", got.Index)
	}
}

func TestBestMatch_FuzzyTitle(t *testing.T) {
	windows := []Window{
		{AppName: "Ghostty", Title: "~/other — zsh", Index: 1},
		{AppName: "Ghostty", Title: "~/myproject — zsh", Index: 2},
	}
	s := discovery.Session{WorkDir: "/Users/x/myproject"}

	got, ok := bestMatch(windows, s)
	if !ok {
		t.Fatal("expected a match")
	}
	if got.Index != 2 {
		t.Errorf("expected window index 2, got %d", got.Index)
	}
}

func TestBestMatch_FuzzyPreferMoreComponents(t *testing.T) {
	windows := []Window{
		{AppName: "Ghostty", Title: "myproject — zsh", Index: 1},
		{AppName: "Ghostty", Title: "/Users/x/myproject — zsh", Index: 2},
	}
	s := discovery.Session{WorkDir: "/Users/x/myproject"}

	got, ok := bestMatch(windows, s)
	if !ok {
		t.Fatal("expected a match")
	}
	// Window 2 has the full path in the title, so it scores higher
	if got.Index != 2 {
		t.Errorf("expected window index 2, got %d", got.Index)
	}
}

func TestBestMatch_CaseInsensitive(t *testing.T) {
	windows := []Window{
		{AppName: "Ghostty", Title: "MYPROJECT — zsh", Index: 1},
	}
	s := discovery.Session{WorkDir: "/Users/x/myproject"}

	got, ok := bestMatch(windows, s)
	if !ok {
		t.Fatal("expected a match")
	}
	if got.Index != 1 {
		t.Errorf("expected window index 1, got %d", got.Index)
	}
}

func TestBestMatch_NoMatch(t *testing.T) {
	windows := []Window{
		{AppName: "Ghostty", Title: "~/other — zsh", Index: 1},
		{AppName: "Ghostty", Title: "~/another — zsh", Index: 2},
	}
	s := discovery.Session{WorkDir: "/Users/x/myproject"}

	_, ok := bestMatch(windows, s)
	if ok {
		t.Fatal("expected no match")
	}
}

func TestBestMatch_EmptyWindows(t *testing.T) {
	s := discovery.Session{WorkDir: "/Users/x/myproject"}
	_, ok := bestMatch(nil, s)
	if ok {
		t.Fatal("expected no match for nil windows")
	}
}

func TestBestMatch_EmptyWorkDir(t *testing.T) {
	windows := []Window{
		{AppName: "Ghostty", Title: "zsh", Index: 1},
	}
	s := discovery.Session{WorkDir: ""}

	_, ok := bestMatch(windows, s)
	if ok {
		t.Fatal("expected no match when session has no WorkDir")
	}
}

func TestBestMatch_WorkDirWinsOverTitle(t *testing.T) {
	windows := []Window{
		{AppName: "Ghostty", Title: "myproject — zsh", Index: 1},
		{AppName: "Ghostty", Title: "other — zsh", WorkDir: "/Users/x/myproject", Index: 2},
	}
	s := discovery.Session{WorkDir: "/Users/x/myproject"}

	got, ok := bestMatch(windows, s)
	if !ok {
		t.Fatal("expected a match")
	}
	// WorkDir exact match should win over title fuzzy match
	if got.Index != 2 {
		t.Errorf("expected window index 2 (WorkDir match), got %d", got.Index)
	}
}
