package commands

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// CmdSpeedtest: tries to run speedtest-cli or opens speedtest.net
func CmdSpeedtest(_ []string) string {
	if p, err := exec.LookPath("speedtest-cli"); err == nil {
		cmd := exec.Command(p, "--simple")
		out, err := cmd.CombinedOutput()
		if err == nil {
			return string(out)
		}
		return "speedtest error: " + err.Error()
	}
	// fallback: open website
	if err := runOpen("https://www.speedtest.net/"); err != nil {
		return "speedtest open error: " + err.Error()
	}
	return "Opened speedtest.net in browser."
}

// CmdSysPerf: simple system summary
func CmdSysPerf(_ []string) string {
	info := []string{}
	info = append(info, "0xRootShell — system summary")
	info = append(info, "OS: "+runtime.GOOS)
	info = append(info, "ARCH: "+runtime.GOARCH)
	// CPU count
	info = append(info, "CPUs: "+fmt.Sprint(runtime.NumCPU()))
	// try some platform-specific commands
	if runtime.GOOS == "windows" {
		// try wmic for basic CPU usage (best-effort)
		if p, err := exec.LookPath("wmic"); err == nil {
			if out, err := exec.Command(p, "cpu", "get", "loadpercentage").CombinedOutput(); err == nil {
				info = append(info, "CPU Load (wmic):")
				info = append(info, string(out))
			}
		}
	} else {
		// linux: try uptime and free -h
		if p, err := exec.LookPath("uptime"); err == nil {
			if out, err := exec.Command(p).CombinedOutput(); err == nil {
				info = append(info, "Uptime: "+strings.TrimSpace(string(out)))
			}
		}
		if p, err := exec.LookPath("free"); err == nil {
			if out, err := exec.Command(p, "-h").CombinedOutput(); err == nil {
				info = append(info, "Memory (free -h):")
				info = append(info, string(out))
			}
		}
	}
	return strings.Join(info, "\n")
}
