//go:build darwin

package main

import (
	"os"
	"os/exec"
	"strings"
)

// ensurePath augments PATH with common macOS directories that are missing
// when the app is launched from the Dock (which inherits a minimal PATH).
func ensurePath() {
	// Directories commonly used by Homebrew, user tools, and IDE CLIs.
	extras := []string{
		"/opt/homebrew/bin",
		"/opt/homebrew/sbin",
		"/usr/local/bin",
		"/usr/local/sbin",
	}

	// Try to get the user's full login shell PATH for anything we missed
	// (e.g. tools added via nvm, pyenv, cargo, etc.).
	if shellPath := loginShellPath(); shellPath != "" {
		for _, dir := range strings.Split(shellPath, ":") {
			if dir != "" {
				extras = append(extras, dir)
			}
		}
	}

	current := os.Getenv("PATH")
	existing := make(map[string]bool)
	for _, dir := range strings.Split(current, ":") {
		existing[dir] = true
	}

	var toAdd []string
	for _, dir := range extras {
		if !existing[dir] {
			toAdd = append(toAdd, dir)
			existing[dir] = true
		}
	}

	if len(toAdd) > 0 {
		os.Setenv("PATH", current+":"+strings.Join(toAdd, ":"))
	}
}

// loginShellPath runs the user's login shell to extract their PATH.
func loginShellPath() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	out, err := exec.Command(shell, "-l", "-c", "echo $PATH").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
