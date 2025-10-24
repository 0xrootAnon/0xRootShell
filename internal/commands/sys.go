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
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
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
		return SysLock()
	case "sleep":
		return SysSleep()
	case "off", "shutdown":
		for _, a := range args {
			if a == "--confirm" || a == "-y" {
				return SysShutdown()
			}
		}
		return "sys off: destructive action. append --confirm to actually shutdown."
	case "bootlog":
		return SysBootLog()
	case "update":
		return "sys update: use OS update tool (Windows Settings / apt / dnf / etc.)"
	default:
		return "sys: unknown subcommand"
	}
}

func SysLock() string {
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

func SysSleep() string {
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

func SysShutdown() string {
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

func SysBootLog() string {
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

func CmdSysPerf() string {
	ps := `
$cpu = (Get-Counter '\Processor(_Total)\% Processor Time').CounterSamples.CookedValue
$proc = Get-Process | Sort-Object -Descending CPU | Select-Object -First 5 | ForEach-Object { "{0, -25} {1,6:N1} CPU {2,8:N1} MB" -f $_.ProcessName, $_.CPU, ($_.WorkingSet/1MB) }
$mem = Get-CimInstance -ClassName Win32_OperatingSystem | Select-Object TotalVisibleMemorySize, FreePhysicalMemory
$totalMB = [math]::Round($mem.TotalVisibleMemorySize/1024,1)
$freeMB = [math]::Round($mem.FreePhysicalMemory/1024,1)
$usedMB = $totalMB - $freeMB
$disk = Get-CimInstance -ClassName Win32_LogicalDisk -Filter "DriveType=3" | ForEach-Object { "{0} {1}G free / {2}G total" -f $_.DeviceID, [math]::Round($_.FreeSpace/1GB,1), [math]::Round($_.Size/1GB,1) }
$net = Get-NetAdapter | Where-Object Status -eq 'Up' | Select-Object -First 1 -ExpandProperty Name
$netstats = ""
if ($net) {
  $bytes = (Get-NetAdapterStatistics -Name $net)
  $netstats = "{0}: Tx={1} Rx={2}" -f $net, [math]::Round($bytes.TransmittedBytes/1KB,1), [math]::Round($bytes.ReceivedBytes/1KB,1)
}
"CPU (total %): " + [math]::Round($cpu,1)
"Memory (MB): used " + $usedMB + "  free " + $freeMB + "  total " + $totalMB
"Disk:"
$disk
"Top Processes (CPU then Mem):"
$proc
if ($netstats) { "Network: " + $netstats }
`
	out, err := runPowerShell(ps)
	if err != nil {
		return fmt.Sprintf("sys perf: failed to query performance counters: %s\n(Ensure PowerShell is available and running on Windows)", err.Error())
	}
	return out
}
func runPowerShell(script string) (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := strings.TrimSpace(out.String())
	if err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = err.Error()
		}
		return result, fmt.Errorf("%s", errMsg)
	}
	return result, nil
}

var (
	recLock  sync.Mutex
	recProcs = map[string]*exec.Cmd{}
)

func detectFFmpeg() (string, error) {
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p, nil
	}
	return "", errors.New("ffmpeg not found")
}

func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

func safeCmdStart(cmd *exec.Cmd) error {
	_ = ensureDir("data/recordings/logs")
	outf, err := os.Create(filepath.Join("data/recordings/logs", fmt.Sprintf("proc_%d.out.log", time.Now().UnixNano())))
	if err == nil {
		cmd.Stdout = outf
	}
	errf, err2 := os.Create(filepath.Join("data/recordings/logs", fmt.Sprintf("proc_%d.err.log", time.Now().UnixNano())))
	if err2 == nil {
		cmd.Stderr = errf
	}
	if err != nil && err2 != nil {
	}
	return cmd.Start()
}

func CmdShowNotifications() string {
	cmd := exec.Command("cmd", "/C", "start", "ms-actioncenter:")
	if err := cmd.Start(); err == nil {
		return "Opened Action Center. Note: Windows does not allow enumerating other apps' notifications programmatically for security reasons. This opens the notification center where you can view them."
	}
	_, err := runPowerShell("Start-Process -FilePath 'ms-actioncenter:'")
	if err == nil {
		return "Opened Action Center."
	}
	return "Failed to open Action Center. Try opening notifications manually (Win+A)."
}
