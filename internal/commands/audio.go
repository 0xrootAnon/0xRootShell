//go:build !windows
// +build !windows

package commands

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func CmdAudio(args []string) string {
	if len(args) == 0 {
		return "audio: expected subcommand: vol <0-100> | mute | unmute"
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "mute":
		return audioMute(true)
	case "unmute":
		return audioMute(false)
	case "vol", "volume":
		if len(args) < 2 {
			return "audio vol: expected percentage 0-100, e.g. `audio vol 40`"
		}
		pct, err := strconv.Atoi(args[1])
		if err != nil || pct < 0 || pct > 100 {
			return "audio vol: value must be an integer 0-100"
		}
		return audioSetVolume(pct)
	default:
		return "audio: unknown subcommand. Try `audio vol <0-100>`, `audio mute`, or `audio unmute`."
	}
}

func audioMute(mute bool) string {
	switch runtime.GOOS {
	case "windows":
		if p, _ := exec.LookPath("nircmd"); p != "" {
			arg := "mutesysvolume"
			val := "1"
			if !mute {
				val = "0"
			}
			cmd := exec.Command(p, arg, val)
			if out, err := cmd.CombinedOutput(); err != nil {
				return "audio mute error: " + err.Error() + " — " + strings.TrimSpace(string(out))
			}
			if mute {
				return "Audio muted (via nircmd)."
			}
			return "Audio unmuted (via nircmd)."
		}
		return "audio mute: nircmd not found. Download from https://www.nirsoft.net/utils/nircmd.html and put nircmd.exe in PATH."
	case "darwin":
		val := "set volume with output muted"
		if !mute {
			val = "set volume without output muted"
		}
		cmd := exec.Command("osascript", "-e", val)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "audio mute error: " + err.Error() + " — " + strings.TrimSpace(string(out))
		}
		if mute {
			return "Audio muted (macOS)."
		}
		return "Audio unmuted (macOS)."
	default:
		if p, _ := exec.LookPath("pactl"); p != "" {
			action := "set-sink-mute"
			val := "1"
			if !mute {
				val = "0"
			}
			cmd := exec.Command(p, action, "@DEFAULT_SINK@", val)
			if out, err := cmd.CombinedOutput(); err == nil {
				if mute {
					return "Audio muted (pactl)."
				}
				return "Audio unmuted (pactl)."
			} else {
				_ = out
			}
		}
		if p, _ := exec.LookPath("amixer"); p != "" {
			arg := "set"
			val := "mute"
			if !mute {
				val = "unmute"
			}
			cmd := exec.Command(p, "Master", arg, val)
			if out, err := cmd.CombinedOutput(); err == nil {
				if mute {
					return "Audio muted (amixer)."
				}
				return "Audio unmuted (amixer)."
			} else {
				_ = out
			}
		}
		return "audio mute: no supported audio control found (try installing pactl/pulseaudio, amixer, or nircmd on Windows)."
	}
}

func audioSetVolume(pct int) string {
	switch runtime.GOOS {
	case "windows":
		if p, _ := exec.LookPath("nircmd"); p != "" {
			val := int((65535 * pct) / 100)
			cmd := exec.Command(p, "setsysvolume", strconv.Itoa(val))
			if out, err := cmd.CombinedOutput(); err != nil {
				return "audio vol error: " + err.Error() + " — " + strings.TrimSpace(string(out))
			}
			return fmt.Sprintf("Volume set to %d%% (via nircmd).", pct)
		}
		return "audio vol: nircmd not found. Install nircmd and place nircmd.exe in PATH."
	case "darwin":
		script := fmt.Sprintf("set volume output volume %d", pct)
		cmd := exec.Command("osascript", "-e", script)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "audio vol error: " + err.Error() + " — " + strings.TrimSpace(string(out))
		}
		return fmt.Sprintf("Volume set to %d%% (macOS).", pct)
	default:
		if p, _ := exec.LookPath("pactl"); p != "" {
			val := fmt.Sprintf("%d%%", pct)
			cmd := exec.Command(p, "set-sink-volume", "@DEFAULT_SINK@", val)
			if out, err := cmd.CombinedOutput(); err == nil {
				return fmt.Sprintf("Volume set to %d%% (pactl).", pct)
			} else {
				return "audio vol error: " + err.Error() + " — " + strings.TrimSpace(string(out))
			}
		}
		if p, _ := exec.LookPath("amixer"); p != "" {
			val := fmt.Sprintf("%d%%", pct)
			cmd := exec.Command(p, "sset", "Master", val)
			if out, err := cmd.CombinedOutput(); err == nil {
				return fmt.Sprintf("Volume set to %d%% (amixer).", pct)
			} else {
				return "audio vol error: " + err.Error() + " — " + strings.TrimSpace(string(out))
			}
		}
		return "audio vol: no supported audio control found (install pactl/pulseaudio or amixer)."
	}
}
