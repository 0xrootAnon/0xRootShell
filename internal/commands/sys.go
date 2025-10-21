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

package commands

import (
	"os/exec"
	"runtime"
	"strings"
)

func CmdSys(args []string) string {
	if len(args) == 0 {
		return "sys: expected 'sys status|lock|sleep|off|bootlog|update'"
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "status":
		return CmdSysStatus()
	case "lock":
		return sysLock()
	case "sleep":
		return sysSleep()
	case "off", "shutdown":
		for _, a := range args {
			if a == "--confirm" || a == "-y" {
				return sysShutdown()
			}
		}
		return "sys off: destructive action. append --confirm to actually shutdown."
	case "bootlog":
		return sysBootLog()
	case "update":
		return "sys update: use OS update tool (Windows Settings / apt / dnf / etc.)"
	default:
		return "sys: unknown subcommand"
	}
}

func sysLock() string {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("rundll32.exe", "user32.dll,LockWorkStation")
		if err := cmd.Run(); err != nil {
			return "sys lock error: " + err.Error()
		}
		return "Locked workstation."
	case "darwin":
		cmd := exec.Command("/System/Library/CoreServices/Menu Extras/User.menu/Contents/Resources/CGSession", "-suspend")
		if err := cmd.Run(); err != nil {
			return "sys lock error: " + err.Error()
		}
		return "Locked screen."
	default:
		if err := exec.Command("loginctl", "lock-session").Run(); err == nil {
			return "Locked session."
		}
		if err := exec.Command("gnome-screensaver-command", "-l").Run(); err == nil {
			return "Locked screen."
		}
		return "sys lock: no known lock command available on this Linux system."
	}
}

func sysSleep() string {
	switch runtime.GOOS {
	case "windows":
		ps := `Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.Application]::SetSuspendState('Suspend', $false, $false)`
		cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
		if err := cmd.Run(); err != nil {
			return "sys sleep error: " + err.Error()
		}
		return "System sleep requested."
	case "darwin":
		cmd := exec.Command("pmset", "sleepnow")
		if err := cmd.Run(); err != nil {
			return "sys sleep error: " + err.Error()
		}
		return "Sleep requested."
	default:
		if err := exec.Command("systemctl", "suspend").Run(); err == nil {
			return "System suspend requested."
		}
		return "sys sleep: need systemctl or permission to suspend."
	}
}

func sysShutdown() string {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("shutdown", "/s", "/t", "0")
		if err := cmd.Start(); err != nil {
			return "sys off error: " + err.Error()
		}
		return "Shutting down..."
	case "darwin":
		cmd := exec.Command("shutdown", "-h", "now")
		if err := cmd.Start(); err != nil {
			return "sys off error: " + err.Error()
		}
		return "Shutting down..."
	default:
		cmd := exec.Command("systemctl", "poweroff")
		if err := cmd.Start(); err != nil {
			return "sys off error: " + err.Error()
		}
		return "Shutting down..."
	}
}

func sysBootLog() string {
	switch runtime.GOOS {
	case "linux":
		out, err := exec.Command("uptime", "-p").CombinedOutput()
		if err == nil {
			return "Uptime: " + strings.TrimSpace(string(out))
		}
		return "bootlog: " + err.Error()
	case "windows":
		out, err := exec.Command("powershell", "-NoProfile", "-Command", "([Management.ManagementDateTimeConverter]::ToDateTime((Get-WmiObject -Class Win32_OperatingSystem).LastBootUpTime)).ToString()").CombinedOutput()
		if err == nil {
			return "Last boot: " + strings.TrimSpace(string(out))
		}
		return "bootlog: " + err.Error()
	case "darwin":
		out, err := exec.Command("sysctl", "-n", "kern.boottime").CombinedOutput()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
		return "bootlog: " + err.Error()
	default:
		return "bootlog: unsupported OS"
	}
}
