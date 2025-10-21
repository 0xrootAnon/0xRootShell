//go:build windows
// +build windows

package commands

import (
	"fmt"
	"os/exec"
	"strings"
)

// CmdNet is the entry point engine expects for "net" commands on Windows.
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

// wifiList tries netsh first, then falls back to PowerShell.
func wifiList() string {
	// Try netsh wlan show interfaces (preferred)
	if out, err := exec.Command("netsh", "wlan", "show", "interfaces").CombinedOutput(); err == nil {
		return sanitizeOutput(strings.TrimSpace(string(out)))
	} else {
		// even if err != nil, return sanitized output rather than failing the program
		if len(out) > 0 {
			return sanitizeOutput(strings.TrimSpace(string(out)))
		}
	}

	// Fallback to PowerShell Get-NetAdapter table if netsh didn't produce output
	if out2, err2 := exec.Command("powershell", "-NoProfile", "-Command", "Get-NetAdapter | Format-Table -Auto").CombinedOutput(); err2 == nil {
		return sanitizeOutput(strings.TrimSpace(string(out2)))
	} else {
		if len(out2) > 0 {
			return sanitizeOutput(strings.TrimSpace(string(out2)))
		}
	}

	return "wifi list: failed to query adapters (requires netsh or PowerShell)."
}

// wifiToggle attempts to enable/disable adapters named like "*Wi-Fi*".
func wifiToggle(enable bool) string {
	action := "Enable-NetAdapter"
	if !enable {
		action = "Disable-NetAdapter"
	}

	// PowerShell wrapper: try the action, suppress errors, then print adapter status.
	// We force a zero exit so Go won't treat non-zero exit as an unhandled failure.
	ps := fmt.Sprintf(`
try {
  $a = Get-NetAdapter -Name '*Wi-Fi*' -ErrorAction SilentlyContinue
  if ($a) {
    $a | %s -Confirm:$false -ErrorAction SilentlyContinue
  } else {
    Write-Output 'No adapter matching ''*Wi-Fi*'' found'
  }
} catch {
  Write-Output $_
}
# then print status
Get-NetAdapter -Name '*Wi-Fi*' -ErrorAction SilentlyContinue | Select-Object Name,Status | Out-String
exit 0
`, action)

	out, _ := exec.Command("powershell", "-NoProfile", "-Command", ps).CombinedOutput()
	clean := sanitizeOutput(strings.TrimSpace(string(out)))

	// detect permission/elevation issues in common phrasing
	l := strings.ToLower(clean)
	if strings.Contains(l, "access is denied") || strings.Contains(l, "requires elevation") || strings.Contains(l, "requires administrator") {
		return "wifi toggle failed: requires Administrator privileges. Run 0xRootShell elevated (right-click → Run as administrator) and try again."
	}

	// if PowerShell told us "No adapter matching", return friendly guidance
	if strings.Contains(l, "no adapter matching") || strings.Contains(l, "no adapter named") || strings.Contains(l, "no adapter") {
		return "Wi-Fi toggle attempted: no adapter named like '*Wi-Fi*' found. Run `Get-NetAdapter` in PowerShell to see exact adapter names."
	}

	// determine textual state for the message
	state := "disabled"
	if enable {
		state = "enabled"
	}

	// If `clean` contains status lines, include them; otherwise a generic message
	if clean == "" {
		return fmt.Sprintf("Wi-Fi %s attempted.", state)
	}
	return fmt.Sprintf("Wi-Fi %s attempted.\n%s", state, clean)
}
