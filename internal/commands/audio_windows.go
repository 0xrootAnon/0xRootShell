//go:build windows
// +build windows

package commands

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/itchyny/volume-go" // add this dependency
)

// CmdAudio supports: audio vol <0-100>, audio mute, audio unmute
func CmdAudio(args []string) string {
	if len(args) == 0 {
		return "audio: expected subcommand e.g. 'audio vol 50' or 'audio mute'"
	}
	sub := strings.ToLower(args[0])

	switch sub {
	case "vol", "volume":
		if len(args) < 2 {
			return "audio vol: expected 0-100"
		}
		n, err := strconv.Atoi(args[1])
		if err != nil {
			return "audio vol: invalid number"
		}
		if n < 0 {
			n = 0
		}
		if n > 100 {
			n = 100
		}
		// Try native Go lib first
		if err := volume.SetVolume(n); err == nil {
			return fmt.Sprintf("Volume set to %d%%", n)
		}
		// fallback to nircmd if installed
		if p, err := exec.LookPath("nircmd.exe"); err == nil {
			abs := int(float64(n) / 100.0 * 65535.0)
			cmd := exec.Command(p, "setsysvolume", strconv.Itoa(abs))
			if err := cmd.Run(); err == nil {
				return fmt.Sprintf("Volume set to %d%% (nircmd)", n)
			}
		}
		// last fallback: instruct user
		return "audio vol: failed to set volume. Install volume-go support or nircmd (https://www.nirsoft.net/utils/nircmd.html)."

	case "mute":
		if err := volume.Mute(); err == nil {
			return "Audio muted"
		}
		if p, err := exec.LookPath("nircmd.exe"); err == nil {
			_ = exec.Command(p, "mutesysvolume", "1").Run()
			return "Audio muted (nircmd)"
		}
		// PowerShell best-effort: not reliable without modules
		_ = exec.Command("powershell", "-Command", "Set-AudioDevice -Mute $true").Run()
		return "Audio mute attempted (PowerShell fallback)."

	case "unmute":
		if err := volume.Unmute(); err == nil {
			return "Audio unmuted"
		}
		if p, err := exec.LookPath("nircmd.exe"); err == nil {
			_ = exec.Command(p, "mutesysvolume", "0").Run()
			return "Audio unmuted (nircmd)"
		}
		_ = exec.Command("powershell", "-Command", "Set-AudioDevice -Mute $false").Run()
		return "Audio unmute attempted (PowerShell fallback)."

	default:
		return "audio: unknown subcommand. Try 'audio vol <0-100>' or 'audio mute'"
	}
}
