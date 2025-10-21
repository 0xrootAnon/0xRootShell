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
	"runtime"
	"time"
)

func CmdSysStatus() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return fmt.Sprintf("System status:\n"+
		"  OS: %s/%s\n"+
		"  CPUs: %d\n"+
		"  Alloc: %d KB\n"+
		"  Sys: %d KB\n"+
		"  Goroutines: %d\n"+
		"  Time: %s\n",
		runtime.GOOS,
		runtime.GOARCH,
		runtime.NumCPU(),
		m.Alloc/1024,
		m.Sys/1024,
		runtime.NumGoroutine(),
		time.Now().Format(time.RFC1123))
}
