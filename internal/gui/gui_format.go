package gui

import (
	"fmt"
	"image/color"
	"os/exec"
	"runtime"
	"strconv"

	"claude-sessions/internal/util"
)

// shortenPath replaces the home directory prefix with "~".
var shortenPath = util.ShortenPath

func statusColor(status string) color.NRGBA {
	switch status {
	case "working":
		return colorWorking
	case "waiting":
		return colorWaiting
	default:
		return colorIdle
	}
}

func statusLabel(status string) string {
	switch status {
	case "working":
		return "Working"
	case "waiting":
		return "Waiting for input"
	default:
		return "Idle"
	}
}

func formatAgentsGUI(count int) string {
	if count > 0 {
		return strconv.Itoa(count) + " running"
	}
	return "none"
}

func formatProjectGUI(workDir string) string {
	p := shortenPath(workDir)
	if p == "" {
		return "unknown"
	}
	return util.TruncateProject(p, 40)
}

func formatPercentGUI(val float64) string {
	return util.FormatPercent(val)
}

// playNotificationSound plays a short system alert sound.
func playNotificationSound() {
	switch runtime.GOOS {
	case "darwin":
		// Best-effort: sound playback is non-critical; failure (e.g., missing
		// sound file or no audio device) should not affect the dashboard.
		_ = exec.Command("afplay", "/System/Library/Sounds/Glass.aiff").Run()
	case "linux":
		// Try PulseAudio bell, fall back to BEL character.
		if err := exec.Command("paplay", "/usr/share/sounds/freedesktop/stereo/bell.oga").Run(); err != nil {
			fmt.Print("\a")
		}
	default:
		fmt.Print("\a") // terminal BEL
	}
}
