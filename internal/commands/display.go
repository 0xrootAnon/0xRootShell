package commands

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// CmdDisplay handles brightness control:
//
//	display bright <0-100>
func CmdDisplay(args []string) string {
	if len(args) == 0 {
		return "display: expected subcommand 'bright <0-100>'"
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "bright", "brightness":
		if len(args) < 2 {
			return "display bright: expected value 0-100"
		}
		v, err := strconv.Atoi(args[1])
		if err != nil || v < 0 || v > 100 {
			return "display bright: value must be 0-100"
		}
		return setBrightness(v)
	default:
		return "display: unknown subcommand. Try `display bright <0-100>`."
	}
}

func setBrightness(percent int) string {
	switch runtime.GOOS {
	case "windows":
		// Try PowerShell WMI method (may require admin and only works for some displays)
		script := fmt.Sprintf("(Get-WmiObject -Namespace root/WMI -Class WmiMonitorBrightnessMethods).WmiSetBrightness(1,%d)", percent)
		cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
		if out, err := cmd.CombinedOutput(); err == nil {
			return fmt.Sprintf("Brightness set to %d%% (PowerShell).", percent)
		} else {
			// helpful fallback note
			return "display bright error: " + err.Error() + " — " + strings.TrimSpace(string(out)) + ". If this fails, consider vendor utilities or run with elevated privileges."
		}
	case "darwin":
		// macOS: no standard CLI for brightness; suggest brew install brightness
		if p, _ := exec.LookPath("brightness"); p != "" {
			// brightness utility expects 0..1 float
			val := fmt.Sprintf("%f", float64(percent)/100.0)
			cmd := exec.Command(p, val)
			if out, err := cmd.CombinedOutput(); err == nil {
				return fmt.Sprintf("Brightness set to %d%% (brightness).", percent)
			} else {
				return "display bright error: " + err.Error() + " — " + strings.TrimSpace(string(out))
			}
		}
		return "display bright: macOS requires a helper (try `brew install brightness`) or use System Settings."
	default:
		// Linux: prefer brightnessctl, fallback to xrandr with best-effort
		if p, _ := exec.LookPath("brightnessctl"); p != "" {
			val := fmt.Sprintf("%d%%", percent)
			cmd := exec.Command(p, "set", val)
			if out, err := cmd.CombinedOutput(); err == nil {
				return fmt.Sprintf("Brightness set to %d%% (brightnessctl).", percent)
			} else {
				return "display bright error: " + err.Error() + " — " + strings.TrimSpace(string(out))
			}
		}
		if p, _ := exec.LookPath("xrandr"); p != "" {
			// find primary output via xrandr --query (best-effort)
			out, err := exec.Command(p, "--query").CombinedOutput()
			if err != nil {
				return "display bright error: cannot query displays: " + err.Error()
			}
			lines := strings.Split(string(out), "\n")
			var outName string
			for _, l := range lines {
				if strings.Contains(l, " connected") {
					fields := strings.Fields(l)
					outName = fields[0]
					break
				}
			}
			if outName == "" {
				return "display bright: cannot detect output via xrandr"
			}
			// xrandr brightness is a float 0..1
			f := float64(percent) / 100.0
			cmd := exec.Command(p, "--output", outName, "--brightness", fmt.Sprintf("%f", f))
			if o, err := cmd.CombinedOutput(); err == nil {
				_ = o
				return fmt.Sprintf("Brightness set to %d%% (xrandr on %s).", percent, outName)
			} else {
				return "display bright error: " + err.Error() + " — " + strings.TrimSpace(string(o))
			}
		}
		return "display bright: no supported tool found (install brightnessctl or use xrandr)."
	}
}
