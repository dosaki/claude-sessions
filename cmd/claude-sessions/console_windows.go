//go:build windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

const attachParentProcess = ^uint32(0) // ATTACH_PARENT_PROCESS = (DWORD)-1

var (
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	procAttachConsole = kernel32.NewProc("AttachConsole")
	procGetStdHandle  = kernel32.NewProc("GetStdHandle")
)

func init() {
	// When built with -H windowsgui, there is no console.
	// If the user passed --cli or --once, attach the parent's console
	// so stdout/stderr work as expected.
	if !needsConsole() {
		return
	}

	ret, _, _ := procAttachConsole.Call(uintptr(attachParentProcess))
	if ret == 0 {
		return // no parent console (e.g. double-clicked)
	}

	// Reopen stdout and stderr to the attached console.
	reopenStd(uintptr(int(syscall.STD_OUTPUT_HANDLE)), &os.Stdout, &syscall.Stdout)
	reopenStd(uintptr(int(syscall.STD_ERROR_HANDLE)), &os.Stderr, &syscall.Stderr)
}

// needsConsole scans os.Args for flags that require terminal output.
// This runs in init() before flag.Parse(), so we do a simple string scan.
func needsConsole() bool {
	for _, arg := range os.Args[1:] {
		if arg == "--cli" || arg == "-cli" || arg == "--once" || arg == "-once" {
			return true
		}
	}
	return false
}

func reopenStd(stdHandle uintptr, target **os.File, sysHandle *syscall.Handle) {
	h, _, _ := procGetStdHandle.Call(stdHandle)
	if h == 0 || h == uintptr(syscall.InvalidHandle) {
		return
	}
	*target = os.NewFile(h, "")
	*sysHandle = syscall.Handle(h)
	_ = unsafe.Sizeof(h) // keep unsafe import used
}
