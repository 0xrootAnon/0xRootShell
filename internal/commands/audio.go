package commands

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// CmdAudio handles audio commands: vol <n>, mute, unmute
func CmdAudio(args []string) string {
	if len(args) == 0 {
		return "audio: expected 'vol <0-100>' or 'mute'/'unmute'"
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "mute":
		return audioMute(true)
	case "unmute":
		return audioMute(false)
	case "vol", "volume":
		if len(args) < 2 {
			return "audio vol: expected a value 0-100"
		}
		v, err := strconv.Atoi(args[1])
		if err != nil || v < 0 || v > 100 {
			return "audio vol: value must be 0-100"
		}
		return audioSetVolume(v)
	default:
		// allow shorthand: audio 50
		if n, err := strconv.Atoi(sub); err == nil {
			if n < 0 || n > 100 {
				return "audio: value must be 0-100"
			}
			return audioSetVolume(n)
		}
		return "audio: unknown subcommand"
	}
}

func audioMute(m bool) string {
	switch runtime.GOOS {
	case "linux":
		// try amixer
		if err := exec.Command("amixer", "-D", "pulse", "sset", "Master", "mute").Run(); err == nil && m {
			return "Muted."
		}
		if !m {
			if err := exec.Command("amixer", "-D", "pulse", "sset", "Master", "unmute").Run(); err == nil {
				return "Unmuted."
			}
		}
		return "audio mute/unmute: failed. try installing `amixer` or use your desktop mixer."
	case "windows":
		// Recommend nircmd if available
		if m {
			if err := exec.Command("nircmd.exe", "mutesysvolume", "1").Run(); err == nil {
				return "Muted."
			}
			return "audio mute: install nircmd (https://www.nirsoft.net/utils/nircmd.html) for command-line volume control."
		}
		if err := exec.Command("nircmd.exe", "mutesysvolume", "0").Run(); err == nil {
			return "Unmuted."
		}
		return "audio unmute: install nircmd for CLI control."
	default:
		return "audio mute: unsupported OS"
	}
}

func audioSetVolume(v int) string {
	switch runtime.GOOS {
	case "linux":
		// using amixer for PulseAudio
		p := fmt.Sprintf("%d%%", v)
		if err := exec.Command("amixer", "-D", "pulse", "sset", "Master", p).Run(); err == nil {
			return fmt.Sprintf("Volume set to %d%%", v)
		}
		return "audio vol: failed. ensure `amixer`/alsa-utils is installed."
	case "windows":
		// recommend nircmd
		// nircmd expects 0-65535 volume so scale
		scaled := int((v * 65535) / 100)
		if err := exec.Command("nircmd.exe", "setsysvolume", fmt.Sprintf("%d", scaled)).Run(); err == nil {
			return fmt.Sprintf("Volume set to %d%% (via nircmd)", v)
		}
		return "audio vol: install nircmd to support Windows CLI volume control."
	default:
		return "audio vol: unsupported OS"
	}
}
