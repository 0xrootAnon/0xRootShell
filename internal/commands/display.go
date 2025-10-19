package commands

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// CmdDisplay handles brightness subcommands
func CmdDisplay(args []string) string {
	if len(args) == 0 {
		return "display: expected 'bright <0-100>'"
	}
	sub := strings.ToLower(args[0])
	if sub == "bright" || sub == "brightness" {
		if len(args) < 2 {
			return "display bright: expected 0-100"
		}
		v, err := strconv.Atoi(args[1])
		if err != nil || v < 0 || v > 100 {
			return "display bright: value must be 0-100"
		}
		return displaySetBrightness(v)
	}
	// accept direct numbers: display 70
	if n, err := strconv.Atoi(sub); err == nil {
		if n < 0 || n > 100 {
			return "display: value must be 0-100"
		}
		return displaySetBrightness(n)
	}
	return "display: unknown subcommand"
}

func displaySetBrightness(v int) string {
	switch runtime.GOOS {
	case "linux":
		// try brightnessctl
		percent := fmt.Sprintf("%d%%", v)
		if err := exec.Command("brightnessctl", "set", percent).Run(); err == nil {
			return fmt.Sprintf("Brightness set to %d%%", v)
		}
		// try xrandr (best-effort)
		// NOTE: xrandr brightness is a multiplier (0.0-1.0)
		// leave user instruction if missing
		return "display bright: install `brightnessctl` or use your DE's brightness control."
	case "windows":
		return "display bright: Windows brightness control requires platform APIs or third-party tools. Try changing brightness via system settings."
	case "darwin":
		if err := exec.Command("osascript", "-e", fmt.Sprintf("tell application \"System Events\" to set the value of the brightness slider of the first display preferences pane to %d", v)).Run(); err == nil {
			return fmt.Sprintf("Brightness set to %d%%", v)
		}
		return "display bright: failed on macOS"
	default:
		return "display bright: unsupported OS"
	}
}
