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
