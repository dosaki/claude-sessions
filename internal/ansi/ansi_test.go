package ansi

import "testing"

func TestVisibleLen(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{"plain", "hello", 5},
		{"empty", "", 0},
		{"bold", Bold + "hello" + Reset, 5},
		{"nested", Bold + Blue + "hi" + Reset, 2},
		{"mixed", "a" + Dim + "b" + Reset + "c", 3},
		{"only_ansi", Bold + Reset, 0},
		{"unicode", "café", 4},
		{"ansi_plus_unicode", Bold + "café" + Reset, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VisibleLen(tt.in); got != tt.want {
				t.Errorf("VisibleLen(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestPad(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		width int
		want  string
	}{
		{"needs_padding", "hi", 5, "hi   "},
		{"exact_width", "hello", 5, "hello"},
		{"over_width", "hello!", 5, "hello!"},
		{"ansi_not_counted", Bold + "hi" + Reset, 5, Bold + "hi" + Reset + "   "},
		{"zero_width", "hi", 0, "hi"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Pad(tt.in, tt.width); got != tt.want {
				t.Errorf("Pad(%q, %d) = %q, want %q", tt.in, tt.width, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"no_truncation", "hello", 10, "hello"},
		{"exact_fit", "hello", 5, "hello"},
		{"truncated", "hello world", 5, "hello" + Reset + "…"},
		{"zero_max", "hello", 0, ""},
		{"negative_max", "hello", -1, ""},
		{"ansi_preserved", Bold + "hello world" + Reset, 5, Bold + "hello" + Reset + "…"},
		{"empty", "", 5, ""},
		{"unicode_truncate", "cafébar", 4, "café" + Reset + "…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Truncate(tt.in, tt.max); got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
			}
		})
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		v, lo, hi, want int
	}{
		{5, 0, 10, 5},
		{-1, 0, 10, 0},
		{15, 0, 10, 10},
		{0, 0, 0, 0},
		{5, 5, 5, 5},
	}
	for _, tt := range tests {
		if got := Clamp(tt.v, tt.lo, tt.hi); got != tt.want {
			t.Errorf("Clamp(%d, %d, %d) = %d, want %d", tt.v, tt.lo, tt.hi, got, tt.want)
		}
	}
}
