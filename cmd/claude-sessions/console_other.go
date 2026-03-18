//go:build !windows

package main

// On non-Windows platforms, no console attachment is needed.
// The binary always has a working stdout/stderr regardless of
// how it was launched.
