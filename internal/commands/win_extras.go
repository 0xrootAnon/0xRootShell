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
//go:build windows
// +build windows

package commands

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//NOTE: this file implements Windows-specific helper commands:
//  - CmdSysPerf        -> "sys perf"
//  - CmdShowNotifications -> "show notifications"
//  - CmdRecord         -> "record <screen|cam> <start|stop>"
//  - CmdClickCam       -> "click cam"
//it uses PowerShell where possible, and ffmpeg / nircmd as optional helpers.

var (
	recLock  sync.Mutex
	recProcs = map[string]*exec.Cmd{}
)

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

func detectFFmpeg() (string, error) {
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p, nil
	}
	return "", errors.New("ffmpeg not found")
}

func ensureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

/*
	func fmtTimestamp() string {
		return time.Now().Format("20060102_150405")
	}
*/
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

/*
	func CmdAudio(args []string) string {
		if len(args) == 0 {
			return "audio: expected subcommand. Examples: audio vol 50 | audio mute | audio mic mute | audio switch output <name>"
		}
		sub := strings.ToLower(args[0])

		switch sub {
		case "vol":
			if len(args) < 2 {
				return "audio vol: expected volume value 0-100"
			}
			v, err := strconv.Atoi(args[1])
			if err != nil || v < 0 || v > 100 {
				return "audio vol: invalid value, expect 0-100"
			}
			if p, _ := exec.LookPath("nircmd"); p != "" {
				val := int((float64(v) / 100.0) * 65535.0)
				cmd := exec.Command(p, "setsysvolume", fmt.Sprintf("%d", val))
				if err := cmd.Run(); err == nil {
					return fmt.Sprintf("audio: set volume to %d%% (via nircmd)", v)
				}
			}
			ps := fmt.Sprintf(`Add-Type -TypeDefinition @"

using System;
using System.Runtime.InteropServices;

	public class Vol {
	  [DllImport("user32.dll")] public static extern IntPtr SendMessageW(IntPtr hWnd, uint Msg, IntPtr wParam, IntPtr lParam);
	}

"@ -PassThru > $null
# Attempt to set volume using COM approach (best-effort)
$vol = %d
# Provide guidance if we couldn't set volume
Write-Output "Requested volume: %d%%. If this did not change, install nircmd (https://www.nirsoft.net/utils/nircmd.html) or the AudioDeviceCmdlets PowerShell module."`, v, v)

			_, err = runPowerShell(ps)
			if err != nil {
				return fmt.Sprintf("audio vol: attempted but failed to set volume automatically: %s\nTry installing nircmd and running: nircmd setsysvolume <value>\n(Use nircmd or AudioDeviceCmdlets for reliable control)", err.Error())
			}
			return fmt.Sprintf("audio: requested set volume to %d%% (PowerShell attempted). If no change, install nircmd or AudioDeviceCmdlets.", v)
		case "mute":
			if p, _ := exec.LookPath("nircmd"); p != "" {
				cmd := exec.Command(p, "mutesysvolume", "2") // toggle
				if err := cmd.Run(); err == nil {
					return "audio: toggled mute (via nircmd)"
				}
			}
			ps := `if (Get-Module -ListAvailable -Name AudioDeviceCmdlets) {
	    Import-Module AudioDeviceCmdlets
	    $d = Get-AudioDevice -Playback
	    if ($d) { $d | Set-AudioDevice -MuteToggle; Write-Output "audio: toggled mute (AudioDeviceCmdlets)" }
	} else { Write-Output "audio: AudioDeviceCmdlets module not installed. Run 'Install-Module -Name AudioDeviceCmdlets -Scope CurrentUser' in an elevated PowerShell to enable advanced audio controls." }`

			out, _ := runPowerShell(ps)
			if strings.TrimSpace(out) != "" {
				return out
			}
			return "audio: unable to toggle mute automatically. Install nircmd or AudioDeviceCmdlets for full control."
		case "mic":
			if len(args) < 2 {
				return "audio mic: expected 'mute' or 'unmute' or 'toggle'"
			}
			act := strings.ToLower(args[1])
			ps := `if (Get-Module -ListAvailable -Name AudioDeviceCmdlets) {
	    Import-Module AudioDeviceCmdlets
	    $d = Get-AudioDevice -Capture
	    if ($d) {
	        $d | Set-AudioDevice -MuteToggle
	        Write-Output "audio mic: toggled capture device mute (AudioDeviceCmdlets)"
	    }
	} else {

	    Write-Output "audio mic: AudioDeviceCmdlets module not installed. Run 'Install-Module -Name AudioDeviceCmdlets -Scope CurrentUser' to enable."
	}`

			outPS, _ := runPowerShell(ps)
			if strings.TrimSpace(outPS) != "" {
				if act == "mute" || act == "unmute" {
					return outPS + "\nNote: the current backend toggled mute. For explicit mute/unmute support install AudioDeviceCmdlets."
				}
				return outPS
			}
			if act == "toggle" {
				return "audio mic: attempted to toggle microphone mute. Install AudioDeviceCmdlets for reliable control."
			}
			return "audio mic: explicit mute/unmute not available with installed tools. Install the PowerShell module AudioDeviceCmdlets or use a third-party utility (e.g., nircmd) and retry."
		case "switch":
			if len(args) < 3 {
				return "audio switch: expected 'output <device-name>'"
			}
			which := strings.ToLower(args[1])
			if which != "output" {
				return "audio switch: expected 'audio switch output <device>'"
			}
			deviceName := strings.Join(args[2:], " ")
			ps := fmt.Sprintf(`if (Get-Module -ListAvailable -Name AudioDeviceCmdlets) {
	    Import-Module AudioDeviceCmdlets
	    $d = Get-AudioDevice -List | Where-Object { $_.Name -like "*%s*" -or $_.DeviceId -like "*%s*" } | Select-Object -First 1
	    if ($d) {
	        Set-DefaultAudioDevice -Index $d.Index
	        Write-Output "audio: switched default output to " + $d.Name
	    } else {
	        Write-Output "audio: device '%s' not found (try a partial name). Run 'Get-AudioDevice -List' to enumerate."
	    }
	} else {

	    Write-Output "audio: AudioDeviceCmdlets not installed. Run 'Install-Module -Name AudioDeviceCmdlets -Scope CurrentUser' in PowerShell as admin."
	}`, escapeForPS(deviceName), escapeForPS(deviceName), escapeForPS(deviceName))

			out, _ := runPowerShell(ps)
			if strings.TrimSpace(out) != "" {
				return out
			}
			return "audio switch: cannot switch output automatically. Install AudioDeviceCmdlets PowerShell module or nircmd and retry."
		default:
			return "audio: unknown subcommand. Try 'audio vol <0-100>' or 'audio mute' or 'audio mic mute' or 'audio switch output <name>'"
		}
	}
*/
func escapeForPS(s string) string {
	return strings.ReplaceAll(s, `'`, `''`)
}

/*func CmdRecord(args []string) string {
	if len(args) < 2 {
		return "record: expected 'record screen start|stop' or 'record cam start|stop'"
	}
	target := strings.ToLower(args[0])
	action := strings.ToLower(args[1])

	if action != "start" && action != "stop" {
		return "record: expected start or stop"
	}

	switch target {
	case "screen":
		if action == "start" {
			return startFFmpegScreen()
		}
		return stopRecording("screen")
	case "cam":
		if action == "start" {
			return startFFmpegCam()
		}
		return stopRecording("cam")
	default:
		return "record: unknown target. Use 'screen' or 'cam'"
	}
}
*/
/*func startFFmpegScreen() string {
	ff, err := detectFFmpeg()
	if err != nil {
		return "record: ffmpeg not found. Install ffmpeg to enable recording (https://ffmpeg.org/download.html)."
	}
	_ = ensureDir("data/recordings")
	outFile := filepath.Join("data", "recordings", fmt.Sprintf("screen_%s.mp4", fmtTimestamp()))

	args := []string{"-y", "-f", "gdigrab", "-framerate", "30", "-i", "desktop", "-vcodec", "libx264", "-preset", "veryfast", "-crf", "23", outFile}
	cmd := exec.Command(ff, args...)
	recLock.Lock()
	defer recLock.Unlock()
	if _, exists := recProcs["screen"]; exists {
		return "record: screen recording already running"
	}
	if err := safeCmdStart(cmd); err != nil {
		return fmt.Sprintf("record: failed to start ffmpeg: %v", err)
	}
	recProcs["screen"] = cmd
	return fmt.Sprintf("Recording started (screen). Output: %s\nTo stop: type 'record screen stop'", outFile)
}

func startFFmpegCam() string {
	ff, err := detectFFmpeg()
	if err != nil {
		return "record: ffmpeg not found. Install ffmpeg to enable recording (https://ffmpeg.org/download.html)."
	}
	listCmd := exec.Command(ff, "-list_devices", "true", "-f", "dshow", "-i", "dummy")
	var stderr bytes.Buffer
	listCmd.Stderr = &stderr
	_ = listCmd.Run()
	out := stderr.String()
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindAllStringSubmatch(out, -1)
	var videoDevice string
	for _, m := range matches {
		if len(m) > 1 {
			videoDevice = m[1]
			break
		}
	}
	if videoDevice == "" {
		return "record cam: could not detect camera device via ffmpeg. Run 'ffmpeg -list_devices true -f dshow -i dummy' in a terminal to see available devices."
	}
	_ = ensureDir("data/recordings")
	outFile := filepath.Join("data", "recordings", fmt.Sprintf("cam_%s.mp4", fmtTimestamp()))
	args := []string{"-y", "-f", "dshow", "-i", fmt.Sprintf("video=%s", videoDevice), "-vcodec", "libx264", "-preset", "veryfast", "-crf", "23", outFile}
	cmd := exec.Command(ff, args...)
	recLock.Lock()
	defer recLock.Unlock()
	if _, exists := recProcs["cam"]; exists {
		return "record: cam recording already running"
	}
	if err := safeCmdStart(cmd); err != nil {
		return fmt.Sprintf("record: failed to start ffmpeg: %v", err)
	}
	recProcs["cam"] = cmd
	return fmt.Sprintf("Recording started (cam: %s). Output: %s\nTo stop: type 'record cam stop'", videoDevice, outFile)
}

func stopRecording(key string) string {
	recLock.Lock()
	defer recLock.Unlock()
	cmd, ok := recProcs[key]
	if !ok || cmd == nil {
		return fmt.Sprintf("record: no %s recording currently running", key)
	}
	if cmd.Process != nil {
		if err := cmd.Process.Kill(); err == nil {
			delete(recProcs, key)
			return fmt.Sprintf("Stopped %s recording (process killed).", key)
		}
		_ = cmd.Process.Kill()
		delete(recProcs, key)
		return fmt.Sprintf("Stopped %s recording (process killed).", key)
	}
	delete(recProcs, key)
	return fmt.Sprintf("record: no running process found for %s", key)
}

func CmdClickCam() string {
	ff, err := detectFFmpeg()
	if err != nil {
		return "click cam: ffmpeg not found. Install ffmpeg to enable camera snapshots (https://ffmpeg.org/download.html)."
	}
	return CmdClickCamPS()
	listCmd := exec.Command(ff, "-list_devices", "true", "-f", "dshow", "-i", "dummy")
	var stderr bytes.Buffer
	listCmd.Stderr = &stderr
	_ = listCmd.Run()
	out := stderr.String()
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindAllStringSubmatch(out, -1)
	var videoDevice string
	for _, m := range matches {
		if len(m) > 1 {
			videoDevice = m[1]
			break
		}
	}
	if videoDevice == "" {
		return "click cam: could not detect camera device via ffmpeg. Run 'ffmpeg -list_devices true -f dshow -i dummy' to enumerate devices."
	}
	_ = ensureDir("data/recordings")
	outFile := filepath.Join("data", "recordings", fmt.Sprintf("cam_snap_%s.png", fmtTimestamp()))
	args := []string{"-y", "-f", "dshow", "-i", fmt.Sprintf("video=%s", videoDevice), "-frames:v", "1", outFile}
	cmd := exec.Command(ff, args...)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return fmt.Sprintf("click cam: ffmpeg error: %s", errb.String())
	}
	_ = exec.Command("cmd", "/C", "start", "", outFile).Start()
	return fmt.Sprintf("Saved snapshot to %s and opened it.", outFile)
}
*/
