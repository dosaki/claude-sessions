package render

import (
	"strings"
	"testing"

	"claude-sessions/internal/ansi"
	"claude-sessions/internal/discovery"
)

func TestShortenPath(t *testing.T) {
	origHome := discovery.HomeDir
	defer func() { discovery.HomeDir = origHome }()
	discovery.HomeDir = "/home/testuser"

	tests := []struct {
		name string
		path string
		want string
	}{
		{"home_prefix", "/home/testuser/dev/app", "~/dev/app"},
		{"exact_home", "/home/testuser", "~"},
		{"no_match", "/var/log/syslog", "/var/log/syslog"},
		{"empty", "", ""},
		{"partial_match", "/home/testuser2/dev", "/home/testuser2/dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shortenPath(tt.path); got != tt.want {
				t.Errorf("shortenPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestShortenPath_NoHome(t *testing.T) {
	origHome := discovery.HomeDir
	defer func() { discovery.HomeDir = origHome }()
	discovery.HomeDir = ""

	if got := shortenPath("/home/user/dev"); got != "/home/user/dev" {
		t.Errorf("shortenPath with empty HomeDir = %q, want unchanged", got)
	}
}

func TestFormatStatusDisplay(t *testing.T) {
	tests := []struct {
		status      string
		wantVisible string
	}{
		{"working", "● Working"},
		{"waiting", "● Waiting for input"},
		{"idle", "● Idle"},
		{"unknown", "● Idle"}, // default case
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := formatStatusDisplay(tt.status)
			visible := ansi.Re.ReplaceAllString(got, "")
			if visible != tt.wantVisible {
				t.Errorf("formatStatusDisplay(%q) visible = %q, want %q", tt.status, visible, tt.wantVisible)
			}
		})
	}
}

func TestFormatAgents(t *testing.T) {
	tests := []struct {
		count       int
		wantVisible string
	}{
		{0, "none"},
		{1, "1 running"},
		{5, "5 running"},
	}
	for _, tt := range tests {
		t.Run(tt.wantVisible, func(t *testing.T) {
			got := formatAgents(tt.count)
			visible := ansi.Re.ReplaceAllString(got, "")
			if visible != tt.wantVisible {
				t.Errorf("formatAgents(%d) visible = %q, want %q", tt.count, visible, tt.wantVisible)
			}
		})
	}
}

func TestSummaryLine(t *testing.T) {
	tests := []struct {
		name                         string
		total, working, waiting, idle int
		wantContains                 []string
		wantNotContains              []string
	}{
		{
			"all_states",
			5, 2, 1, 2,
			[]string{"5 session(s)", "2 working", "1 waiting", "2 idle"},
			nil,
		},
		{
			"only_working",
			3, 3, 0, 0,
			[]string{"3 session(s)", "3 working"},
			[]string{"waiting", "idle"},
		},
		{
			"only_idle",
			1, 0, 0, 1,
			[]string{"1 session(s)", "1 idle"},
			[]string{"working", "waiting"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summaryLine(tt.total, tt.working, tt.waiting, tt.idle)
			visible := ansi.Re.ReplaceAllString(got, "")
			for _, want := range tt.wantContains {
				if !strings.Contains(visible, want) {
					t.Errorf("summaryLine() visible = %q, missing %q", visible, want)
				}
			}
			for _, notWant := range tt.wantNotContains {
				if strings.Contains(visible, notWant) {
					t.Errorf("summaryLine() visible = %q, should not contain %q", visible, notWant)
				}
			}
		})
	}
}
