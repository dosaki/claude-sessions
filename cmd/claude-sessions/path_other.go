//go:build !darwin

package main

// ensurePath is a no-op on non-macOS platforms, where apps typically
// inherit the full shell PATH regardless of launch method.
func ensurePath() {}
