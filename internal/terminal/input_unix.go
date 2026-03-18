//go:build !windows

package terminal

import (
	"os"
	"os/exec"
	"strings"
)

var savedTermState string

// EnableRawMode puts stdin into character-at-a-time mode with echo disabled,
// so individual keypresses can be read without waiting for Enter.
func EnableRawMode() error {
	cmd := exec.Command("stty", "-g")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return err
	}
	savedTermState = strings.TrimSpace(string(out))

	raw := exec.Command("stty", "-icanon", "-echo")
	raw.Stdin = os.Stdin
	return raw.Run()
}

// DisableRawMode restores the terminal to its previous state.
func DisableRawMode() {
	if savedTermState == "" {
		return
	}
	cmd := exec.Command("stty", savedTermState)
	cmd.Stdin = os.Stdin
	cmd.Run()
	savedTermState = ""
}

// ReadKeys reads keypresses from stdin and sends them on ch.
// Must be called after EnableRawMode. Blocks until stdin is closed.
func ReadKeys(ch chan<- Key) {
	buf := make([]byte, 16)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil || n == 0 {
			return
		}
		data := buf[:n]
		for len(data) > 0 {
			if data[0] == '\x1b' && len(data) >= 3 && data[1] == '[' {
				switch data[2] {
				case 'A':
					ch <- KeyUp
				case 'B':
					ch <- KeyDown
				}
				data = data[3:]
			} else {
				switch data[0] {
				case '\r', '\n':
					ch <- KeyEnter
				case 'q', 'Q':
					ch <- KeyQuit
				case 'j':
					ch <- KeyDown
				case 'k':
					ch <- KeyUp
				}
				data = data[1:]
			}
		}
	}
}
