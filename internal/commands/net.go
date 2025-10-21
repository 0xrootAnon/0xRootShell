// 0xRootShell — A minimalist, aesthetic terminal for creators
// Copyright (c) 2025 Khwahish Sharma (aka 0xRootAnon)
//
// Licensed under the GNU General Public License v3.0 or later (GPLv3+).
// You may obtain a copy of the License at
// https://www.gnu.org/licenses/gpl-3.0.html
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.

//go:build !windows
// +build !windows

package commands

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func CmdNet(args []string) string {
	if len(args) == 0 {
		return "net: expected subcommand, e.g. `net wifi list|on|off`"
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "wifi", "wireless":
		if len(args) < 2 {
			return "net wifi: expected `list`, `on`, or `off`"
		}
		op := strings.ToLower(args[1])
		switch op {
		case "list":
			return wifiList()
		case "on":
			return wifiToggle(true)
		case "off":
			return wifiToggle(false)
		default:
			return "net wifi: unknown op. Use list|on|off"
		}
	default:
		return "net: unknown subcommand. Try `net wifi list|on|off`"
	}
}

func wifiList() string {
	switch runtime.GOOS {
	case "windows":
		if p, _ := exec.LookPath("netsh"); p != "" {
			out, err := exec.Command(p, "wlan", "show", "networks").CombinedOutput()
			if err != nil {
				return "wifi list error: " + err.Error() + " — " + strings.TrimSpace(string(out))
			}
			return string(out)
		}
		return "wifi list: netsh not found on PATH (Windows)."
	case "darwin":
		if _, err := exec.LookPath("/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport"); err == nil {
			out, err := exec.Command("/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport", "-s").CombinedOutput()
			if err != nil {
				return "wifi list error: " + err.Error()
			}
			return string(out)
		}
		return "wifi list: airport tool not available."
	default:
		if p, _ := exec.LookPath("nmcli"); p != "" {
			out, err := exec.Command(p, "device", "wifi", "list").CombinedOutput()
			if err != nil {
				return "wifi list error: " + err.Error() + " — " + strings.TrimSpace(string(out))
			}
			return string(out)
		}
		if p, _ := exec.LookPath("iwlist"); p != "" {
			out, err := exec.Command(p, "scan").CombinedOutput()
			if err != nil {
				return "wifi list error: " + err.Error() + " — " + strings.TrimSpace(string(out))
			}
			return string(out)
		}
		return "wifi list: no wifi tool found (install NetworkManager/nmcli or iwlist)."
	}
}

func wifiToggle(on bool) string {
	switch runtime.GOOS {
	case "windows":
		state := map[bool]string{true: "Enabled", false: "Disabled"}[on]
		ps := fmt.Sprintf("try { $a = Get-NetAdapter -Name 'Wi-Fi' -ErrorAction SilentlyContinue; if ($a) { Set-NetAdapter -Name $a.Name -Admin %s -Confirm:$false; 'OK' } else { Write-Output 'No adapter named Wi-Fi found' } } catch { Write-Error $_ }", state)
		power := exec.Command("powershell", "-NoProfile", "-Command", ps)
		if out, err := power.CombinedOutput(); err == nil {
			if strings.Contains(string(out), "No adapter named Wi-Fi found") {
				return "wifi toggle: no adapter named 'Wi-Fi' found. Check adapter name or use Windows tools."
			}
			return fmt.Sprintf("Wi-Fi %s (PowerShell).", map[bool]string{true: "enabled", false: "disabled"}[on])
		} else {
			return "wifi toggle error (PowerShell): " + err.Error() + " — " + strings.TrimSpace(string(out))
		}
	case "darwin":
		if p, _ := exec.LookPath("networksetup"); p != "" {
			state := "on"
			if !on {
				state = "off"
			}
			cmd := exec.Command(p, "-setairportpower", "en0", state)
			if out, err := cmd.CombinedOutput(); err == nil {
				return fmt.Sprintf("Wi-Fi %s (networksetup).", state)
			} else {
				return "wifi toggle error: " + err.Error() + " — " + strings.TrimSpace(string(out))
			}
		}
		return "wifi toggle: networksetup tool not found."
	default:
		if p, _ := exec.LookPath("nmcli"); p != "" {
			state := "on"
			if !on {
				state = "off"
			}
			cmd := exec.Command(p, "radio", "wifi", state)
			if out, err := cmd.CombinedOutput(); err == nil {
				return fmt.Sprintf("Wi-Fi %s (nmcli).", state)
			} else {
				return "wifi toggle error: " + err.Error() + " — " + strings.TrimSpace(string(out))
			}
		}
		return "wifi toggle: nmcli (NetworkManager) not found. Use your distro's tools or install NetworkManager."
	}
}
