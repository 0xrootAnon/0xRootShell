//go:build windows
// +build windows

package commands

import (
	"fmt"
	"os/exec"
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

// netWifi handles wifi list/on/off
func netWifi(args []string) string {
	if len(args) == 0 {
		return "net wifi: expected 'list'|'on'|'off'"
	}
	cmd := strings.ToLower(args[0])
	switch cmd {
	case "list":
		// use netsh to list wireless interfaces
		out, err := exec.Command("netsh", "wlan", "show", "interfaces").CombinedOutput()
		if err == nil {
			return string(out)
		}
		// fallback to PowerShell network adapter query
		out2, err2 := exec.Command("powershell", "-Command", "Get-NetAdapter | Format-Table -Auto").CombinedOutput()
		if err2 == nil {
			return string(out2)
		}
		return "net wifi list: error: " + err.Error()
	case "off":
		// disable Wi-Fi adapters (best-effort)
		// Use netsh to set interface state
		out, err := exec.Command("powershell", "-Command", "Get-NetAdapter -Name '*Wi-Fi*' -ErrorAction SilentlyContinue | Disable-NetAdapter -Confirm:$false").CombinedOutput()
		if err != nil {
			return "wifi disable attempted: " + err.Error() + "\n" + string(out)
		}
		return "Wi-Fi disabled (attempted)."
	case "on":
		out, err := exec.Command("powershell", "-Command", "Get-NetAdapter -Name '*Wi-Fi*' -ErrorAction SilentlyContinue | Enable-NetAdapter -Confirm:$false").CombinedOutput()
		if err != nil {
			return "wifi enable attempted: " + err.Error() + "\n" + string(out)
		}
		return "Wi-Fi enabled (attempted)."
	default:
		return "net wifi: unknown subcommand"
	}
}

// wifiList uses netsh (preferred) and falls back to PowerShell when necessary.
func wifiList() string {
	// Try netsh wlan show interfaces
	if out, err := exec.Command("netsh", "wlan", "show", "interfaces").CombinedOutput(); err == nil {
		return strings.TrimSpace(string(out))
	}
	// Fallback: PowerShell Get-NetAdapter and Get-NetConnectionProfile
	if out, err := exec.Command("powershell", "-Command", "Get-NetAdapter | Format-Table -Auto").CombinedOutput(); err == nil {
		return strings.TrimSpace(string(out))
	}
	return "wifi list: failed to query adapters (requires netsh or PowerShell)."
}

// wifiToggle enables/disables Wi-Fi adapters named like "*Wi-Fi*" (best-effort).
func wifiToggle(enable bool) string {
	action := "Enable-NetAdapter"
	if !enable {
		action = "Disable-NetAdapter"
	}
	// Use PowerShell to enable/disable adapters matching Wi-Fi
	psCmd := "Get-NetAdapter -Name '*Wi-Fi*' -ErrorAction SilentlyContinue | " + action + " -Confirm:$false"
	out, err := exec.Command("powershell", "-Command", psCmd).CombinedOutput()
	if err == nil {
		return fmt.Sprintf("Wi-Fi %s attempted.\n%s", ternary(enable, "enabled", "disabled"), strings.TrimSpace(string(out)))
	}
	// Last-ditch: try netsh interface (interface name may differ)
	state := "enabled"
	if !enable {
		state = "disabled"
	}
	if out2, err2 := exec.Command("netsh", "interface", "set", "interface", "name=\"Wi-Fi\"", "admin="+state).CombinedOutput(); err2 == nil {
		return fmt.Sprintf("Wi-Fi %s attempted (netsh).", ternary(enable, "enabled", "disabled")) + "\n" + strings.TrimSpace(string(out2))
	}
	return fmt.Sprintf("wifi toggle: failed to %s Wi-Fi. PowerShell/netsh attempts failed: %v / %v", ternary(enable, "enable", "disable"), err, nil)
}

// small helper
func ternary(b bool, a, c string) string {
	if b {
		return a
	}
	return c
}
