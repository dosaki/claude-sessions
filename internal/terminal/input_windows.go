//go:build windows

package terminal

// EnableRawMode is a no-op on Windows (TODO: implement via SetConsoleMode).
func EnableRawMode() error { return nil }

// DisableRawMode is a no-op on Windows.
func DisableRawMode() {}

// ReadKeys is a no-op on Windows (TODO: implement via ReadConsoleInput).
func ReadKeys(ch chan<- Key) {}
