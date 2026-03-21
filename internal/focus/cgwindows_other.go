//go:build !darwin && !windows

package focus

type cgWindowInfo struct {
	PID          int
	WindowNumber int
	Owner        string
	Title        string
}

func listCGWindows(targetPID int) []cgWindowInfo {
	return nil
}
