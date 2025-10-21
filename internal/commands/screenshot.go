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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

func CmdScreenshot(args []string) string {
	dir := filepath.Join("data", "screenshots")
	_ = os.MkdirAll(dir, 0755)
	fname := filepath.Join(dir, fmt.Sprintf("shot-%d.png", time.Now().Unix()))
	switch runtime.GOOS {
	case "linux":
		if err := exec.Command("gnome-screenshot", "-f", fname).Run(); err == nil {
			return "Saved screenshot: " + fname
		}
		if err := exec.Command("scrot", fname).Run(); err == nil {
			return "Saved screenshot: " + fname
		}
		return "screenshot: install gnome-screenshot or scrot"
	case "windows":
		ps := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms;Add-Type -AssemblyName System.Drawing;$bmp = New-Object System.Drawing.Bitmap([Windows.Forms.SystemInformation]::VirtualScreen.Width,[Windows.Forms.SystemInformation]::VirtualScreen.Height);$g = [System.Drawing.Graphics]::FromImage($bmp);$g.CopyFromScreen([Windows.Forms.SystemInformation]::VirtualScreen.X,[Windows.Forms.SystemInformation]::VirtualScreen.Y,0,0,$bmp.Size);$bmp.Save("%s");`, fname)
		cmd := exec.Command("powershell", "-NoProfile", "-Command", ps)
		if err := cmd.Run(); err != nil {
			return "screenshot: failed: " + err.Error()
		}
		return "Saved screenshot: " + fname
	case "darwin":
		if err := exec.Command("screencapture", "-x", fname).Run(); err == nil {
			return "Saved screenshot: " + fname
		}
		return "screenshot: failed on macOS"
	default:
		return "screenshot: unsupported OS"
	}
}
