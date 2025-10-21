//go:build windows
// +build windows

package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Network struct {
	SSID           string
	Authentication string
	Encryption     string
}

var lastNetworks []Network

func CmdNet(args []string) string {
	if len(args) == 0 {
		return "net: expected subcommand, e.g. `net wifi list|on|off|connect|forget|saved`"
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "wifi", "wireless":
		if len(args) < 2 {
			return "net wifi: expected `list`, `on`, `off`, `connect`, `forget`, or `saved`"
		}
		op := strings.ToLower(args[1])
		switch op {
		case "list":
			return wifiList()
		case "on":
			return wifiToggle(true)
		case "off":
			return wifiToggle(false)
		case "connect":
			if len(args) < 3 {
				return "net wifi connect: expected an index (e.g. `net wifi connect 3` or `net wifi connect 3 <password>`)"
			}
			idx, err := strconv.Atoi(args[2])
			if err != nil || idx <= 0 {
				return "net wifi connect: invalid index"
			}
			var pwd string
			if len(args) >= 4 {
				pwd = args[3]
			}
			return wifiConnect(idx-1, pwd)
		case "forget":
			if len(args) < 3 {
				return "net wifi forget: expected an index (e.g. `net wifi forget 2`)"
			}
			idx, err := strconv.Atoi(args[2])
			if err != nil || idx <= 0 {
				return "net wifi forget: invalid index"
			}
			return wifiForget(idx - 1)
		case "saved":
			return wifiSaved()
		default:
			return "net wifi: unknown op. Use list|on|off|connect|forget|saved"
		}
	default:
		return "net: unknown subcommand. Try `net wifi list|on|off|connect|forget|saved`"
	}
}

func wifiList() string {
	out, _ := exec.Command("netsh", "wlan", "show", "networks", "mode=bssid").CombinedOutput()
	clean := sanitizeOutput(strings.TrimSpace(string(out)))
	if clean == "" {
		if out2, _ := exec.Command("netsh", "wlan", "show", "networks").CombinedOutput(); len(out2) > 0 {
			clean = sanitizeOutput(strings.TrimSpace(string(out2)))
		}
	}

	lines := strings.Split(clean, "\n")
	var nets []Network
	var cur Network
	for _, l := range lines {
		line := strings.TrimSpace(l)
		if strings.HasPrefix(line, "SSID ") && strings.Contains(line, ":") {
			if cur.SSID != "" {
				nets = append(nets, cur)
				cur = Network{}
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				cur.SSID = strings.TrimSpace(parts[1])
			}
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "authentication") && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				cur.Authentication = strings.TrimSpace(parts[1])
			}
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "encryption") && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				cur.Encryption = strings.TrimSpace(parts[1])
			}
			continue
		}
	}
	if cur.SSID != "" {
		nets = append(nets, cur)
	}

	lastNetworks = nets

	if len(nets) == 0 {
		if clean != "" {
			return "No networks parsed. Raw output:\n" + clean
		}
		return "No Wi-Fi networks found."
	}

	sb := &strings.Builder{}
	for i, n := range nets {
		fmt.Fprintf(sb, "%d) %s", i+1, n.SSID)
		if n.Authentication != "" || n.Encryption != "" {
			fmt.Fprintf(sb, " — %s / %s", n.Authentication, n.Encryption)
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\nTip: connect with `net wifi connect <number>` or `net wifi connect <number> <password>`\n")
	return sb.String()
}

func wifiSaved() string {
	out, err := exec.Command("netsh", "wlan", "show", "profiles").CombinedOutput()
	clean := sanitizeOutput(strings.TrimSpace(string(out)))
	if err != nil && clean == "" {
		return "Failed to query saved profiles: " + err.Error()
	}

	lines := strings.Split(clean, "\n")
	var profiles []string
	for _, l := range lines {
		line := strings.TrimSpace(l)
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			left := strings.ToLower(strings.TrimSpace(parts[0]))
			right := strings.TrimSpace(parts[1])
			if (strings.Contains(left, "profile") || strings.Contains(left, "all user profile")) && right != "" {
				profiles = append(profiles, right)
			}
		}
	}

	connOut, _ := exec.Command("netsh", "wlan", "show", "interfaces").CombinedOutput()
	connClean := sanitizeOutput(strings.TrimSpace(string(connOut)))
	connectedName := ""
	if connClean != "" {
		for _, l := range strings.Split(connClean, "\n") {
			line := strings.TrimSpace(l)
			if strings.HasPrefix(strings.ToLower(line), "profile") && strings.Contains(line, ":") {
				parts := strings.SplitN(line, ":", 2)
				connectedName = strings.TrimSpace(parts[1])
				break
			}
			if strings.HasPrefix(strings.ToLower(line), "ssid") && strings.Contains(line, ":") {
				parts := strings.SplitN(line, ":", 2)
				connectedName = strings.TrimSpace(parts[1])
			}
		}
	}

	if len(profiles) == 0 {
		if clean != "" {
			return "No saved profiles parsed. Raw output:\n" + clean
		}
		return "No saved Wi-Fi profiles found."
	}

	sb := &strings.Builder{}
	for i, p := range profiles {
		flag := ""
		if connectedName != "" && strings.EqualFold(p, connectedName) {
			flag = " (connected)"
		}
		fmt.Fprintf(sb, "%d) %s%s\n", i+1, p, flag)
	}
	sb.WriteString("\nTip: forget a profile with `net wifi forget <number-from-listed-scan>` (or run netsh commands directly).\n")
	return sb.String()
}

func wifiToggle(enable bool) string {
	action := "Enable-NetAdapter"
	if !enable {
		action = "Disable-NetAdapter"
	}

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

	l := strings.ToLower(clean)
	if strings.Contains(l, "access is denied") || strings.Contains(l, "requires elevation") || strings.Contains(l, "requires administrator") {
		return "wifi toggle failed: requires Administrator privileges. Run 0xRootShell elevated (right-click → Run as administrator) and try again."
	}

	if strings.Contains(l, "no adapter matching") || strings.Contains(l, "no adapter named") || strings.Contains(l, "no adapter") {
		return "Wi-Fi toggle attempted: no adapter named like '*Wi-Fi*' found. Run `Get-NetAdapter` in PowerShell to see exact adapter names."
	}

	state := "disabled"
	if enable {
		state = "enabled"
	}

	if clean == "" {
		return fmt.Sprintf("Wi-Fi %s attempted.", state)
	}
	return fmt.Sprintf("Wi-Fi %s attempted.\n%s", state, clean)
}

func wifiConnect(index int, password string) string {
	if index < 0 || index >= len(lastNetworks) {
		return "net wifi connect: index out of range. Run `net wifi list` first."
	}
	netw := lastNetworks[index]
	ssid := netw.SSID

	profilesOut, _ := exec.Command("netsh", "wlan", "show", "profiles").CombinedOutput()
	profiles := sanitizeOutput(strings.ToLower(strings.TrimSpace(string(profilesOut))))
	if strings.Contains(profiles, strings.ToLower(ssid)) {
		out, err := exec.Command("netsh", "wlan", "connect", "name="+ssid).CombinedOutput()
		clean := sanitizeOutput(strings.TrimSpace(string(out)))
		if err != nil {
			return fmt.Sprintf("Failed to connect to saved profile %s: %v\n%s", ssid, err, clean)
		}
		return fmt.Sprintf("Connecting to saved profile %s...\n%s", ssid, clean)
	}

	if password == "" {
		return fmt.Sprintf("PROMPT_PASSWORD:%d:%s", index, ssid)
	}

	tempDir := filepath.Join("data", "wifi_profiles")
	_ = os.MkdirAll(tempDir, 0755)
	fn := filepath.Join(tempDir, fmt.Sprintf("profile_%d.xml", time.Now().UnixNano()))

	xml := fmt.Sprintf(`<?xml version="1.0"?>
<WLANProfile xmlns="http://www.microsoft.com/networking/WLAN/profile/v1">
  <name>%s</name>
  <SSIDConfig>
    <SSID>
      <name>%s</name>
    </SSID>
  </SSIDConfig>
  <connectionType>ESS</connectionType>
  <connectionMode>auto</connectionMode>
  <MSM>
    <security>
      <authEncryption>
        <authentication>WPA2PSK</authentication>
        <encryption>AES</encryption>
        <useOneX>false</useOneX>
      </authEncryption>
      <sharedKey>
        <keyType>passPhrase</keyType>
        <protected>false</protected>
        <keyMaterial>%s</keyMaterial>
      </sharedKey>
    </security>
  </MSM>
</WLANProfile>`, ssid, ssid, xmlEscape(password))

	if err := os.WriteFile(fn, []byte(xml), 0600); err != nil {
		return "Failed to write temporary profile: " + err.Error()
	}
	defer func() {
		_ = os.Remove(fn)
	}()

	if out, err := exec.Command("netsh", "wlan", "add", "profile", "filename=\""+fn+"\"", "user=current").CombinedOutput(); err != nil {
		clean := sanitizeOutput(strings.TrimSpace(string(out)))
		return fmt.Sprintf("Failed to add profile: %v\n%s", err, clean)
	}

	if out, err := exec.Command("netsh", "wlan", "connect", "name="+ssid).CombinedOutput(); err != nil {
		clean := sanitizeOutput(strings.TrimSpace(string(out)))
		return fmt.Sprintf("Failed to connect to %s: %v\n%s", ssid, err, clean)
	} else {
		clean := sanitizeOutput(strings.TrimSpace(string(out)))
		return fmt.Sprintf("Connecting to %s...\n%s", ssid, clean)
	}
}

func wifiForget(index int) string {
	if index < 0 || index >= len(lastNetworks) {
		return "net wifi forget: index out of range. Run `net wifi list` first."
	}
	ssid := lastNetworks[index].SSID
	out, err := exec.Command("netsh", "wlan", "delete", "profile", "name="+ssid).CombinedOutput()
	clean := sanitizeOutput(strings.TrimSpace(string(out)))
	if err != nil {
		return fmt.Sprintf("Failed to forget profile %s: %v\n%s", ssid, err, clean)
	}
	return fmt.Sprintf("Forgot profile %s.\n%s", ssid, clean)
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
