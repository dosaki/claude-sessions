//go:build !windows

package terminal

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"unsafe"

	"claude-sessions/internal/ansi"
)

type winsize struct {
	Row uint16
	Col uint16
	_   uint16 // xpixel (unused)
	_   uint16 // ypixel (unused)
}

// GetSize returns the current terminal dimensions (cols, rows).
func GetSize() (cols, rows int) {
	var ws winsize
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	)
	if errno != 0 || ws.Col == 0 {
		return defaultCols, defaultRows
	}
	return int(ws.Col), int(ws.Row)
}

// Init sets up platform-specific terminal handling:
// SIGWINCH for resize detection and SIGTERM for graceful shutdown.
func Init() {
	winchCh := make(chan os.Signal, 1)
	signal.Notify(winchCh, syscall.SIGWINCH)
	go func() {
		for range winchCh {
			ResizeRequested.Store(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Print(ansi.CursorShow)
		os.Exit(0)
	}()
}
