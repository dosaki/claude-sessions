//go:build windows

package terminal

import (
	"syscall"
	"time"
	"unsafe"
)

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
	procGetConsoleMode             = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode             = kernel32.NewProc("SetConsoleMode")
)

type coord struct {
	x int16
	y int16
}

type smallRect struct {
	left   int16
	top    int16
	right  int16
	bottom int16
}

type consoleScreenBufferInfo struct {
	size       coord
	cursorPos  coord
	attributes uint16
	window     smallRect
	maxSize    coord
}

// GetSize returns the current terminal dimensions (cols, rows).
func GetSize() (cols, rows int) {
	h, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil {
		return defaultCols, defaultRows
	}
	var info consoleScreenBufferInfo
	r, _, _ := procGetConsoleScreenBufferInfo.Call(
		uintptr(h),
		uintptr(unsafe.Pointer(&info)),
	)
	if r == 0 {
		return defaultCols, defaultRows
	}
	cols = int(info.window.right-info.window.left) + 1
	rows = int(info.window.bottom-info.window.top) + 1
	if cols <= 0 || rows <= 0 {
		return defaultCols, defaultRows
	}
	return cols, rows
}

// enableVirtualTerminal turns on ANSI escape sequence processing in the
// Windows console so that colour codes render correctly.
func enableVirtualTerminal() {
	h, err := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	if err != nil {
		return
	}
	var mode uint32
	r, _, _ := procGetConsoleMode.Call(uintptr(h), uintptr(unsafe.Pointer(&mode)))
	if r == 0 {
		return
	}
	const enableVirtualTerminalProcessing = 0x0004
	procSetConsoleMode.Call(uintptr(h), uintptr(mode|enableVirtualTerminalProcessing))
}

// Init sets up platform-specific terminal handling:
// enables VT processing and polls for terminal size changes.
func Init() {
	enableVirtualTerminal()

	go func() {
		prevCols, prevRows := GetSize()
		for {
			time.Sleep(500 * time.Millisecond)
			cols, rows := GetSize()
			if cols != prevCols || rows != prevRows {
				ResizeRequested.Store(1)
				prevCols, prevRows = cols, rows
			}
		}
	}()
}
