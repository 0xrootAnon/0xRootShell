package commands

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func CmdSpeedtest(_ []string) string {
	if p, err := exec.LookPath("speedtest-cli"); err == nil {
		cmd := exec.Command(p, "--simple")
		out, err := cmd.CombinedOutput()
		if err == nil {
			return string(out)
		}
		return "speedtest error: " + err.Error()
	}
	if err := runOpen("https://www.speedtest.net/"); err != nil {
		return "speedtest open error: " + err.Error()
	}
	return "Opened speedtest.net in browser."
}

func CmdSysPerf(_ []string) string {
	info := []string{}
	info = append(info, "0xRootShell — system summary")
	info = append(info, "OS: "+runtime.GOOS)
	info = append(info, "ARCH: "+runtime.GOARCH)
	info = append(info, "CPUs: "+fmt.Sprint(runtime.NumCPU()))
	if runtime.GOOS == "windows" {
		if p, err := exec.LookPath("wmic"); err == nil {
			if out, err := exec.Command(p, "cpu", "get", "loadpercentage").CombinedOutput(); err == nil {
				info = append(info, "CPU Load (wmic):")
				info = append(info, string(out))
			}
		}
	} else {
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
