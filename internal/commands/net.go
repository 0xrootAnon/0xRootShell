package commands

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// CmdNet handles net wifi list on/off
func CmdNet(args []string) string {
	if len(args) == 0 {
		return "net: expected 'wifi <on|off|list>'"
	}
	if strings.ToLower(args[0]) == "wifi" {
		if len(args) < 2 {
			return "net wifi: expected on/off/list"
		}
		action := strings.ToLower(args[1])
		switch action {
		case "list":
			return wifiList()
		case "on":
			return wifiToggle(true)
		case "off":
			return wifiToggle(false)
		default:
			return "net wifi: expected on/off/list"
		}
	}
	return "net: unknown subcommand"
}

func wifiList() string {
	switch runtime.GOOS {
	case "linux":
		out, err := exec.Command("nmcli", "-t", "-f", "SSID,SECURITY,SIGNAL", "device", "wifi", "list").CombinedOutput()
		if err != nil {
			return "wifi list: " + err.Error()
		}
		return string(out)
	case "windows":
		out, err := exec.Command("netsh", "wlan", "show", "networks", "mode=bssid").CombinedOutput()
		if err != nil {
			return "wifi list: " + err.Error()
		}
		return string(out)
	default:
		return "wifi list: unsupported OS"
	}
}

func wifiToggle(on bool) string {
	switch runtime.GOOS {
	case "linux":
		var cmd *exec.Cmd
		if on {
			cmd = exec.Command("nmcli", "radio", "wifi", "on")
		} else {
			cmd = exec.Command("nmcli", "radio", "wifi", "off")
		}
		if err := cmd.Run(); err != nil {
			return fmt.Sprintf("wifi %v: %v", on, err)
		}
		if on {
			return "Wi-Fi enabled."
		}
		return "Wi-Fi disabled."
	case "windows":
		// use netsh to disable/enable interface named "Wi-Fi"
		name := "Wi-Fi"
		var cmd *exec.Cmd
		if on {
			cmd = exec.Command("netsh", "interface", "set", "interface", name, "admin=ENABLED")
		} else {
			cmd = exec.Command("netsh", "interface", "set", "interface", name, "admin=DISABLED")
		}
		if err := cmd.Run(); err != nil {
			return fmt.Sprintf("wifi %v: %v", on, err)
		}
		if on {
			return "Wi-Fi enabled."
		}
		return "Wi-Fi disabled."
	default:
		return "wifi toggle: unsupported OS"
	}
}
