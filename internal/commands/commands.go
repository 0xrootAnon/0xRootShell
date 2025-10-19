package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/skratchdot/open-golang/open"
)

// opens an application/URL. args joined as command.
func CmdLaunch(args []string) string {
	if len(args) == 0 {
		return "launch: expected an app name, e.g. launch chrome"
	}
	target := strings.Join(args, " ")

	//on win we can use open.Run which maps to "start"
	if err := open.Run(target); err == nil {
		return fmt.Sprintf("Launching %s...", target)
	}

	//fallback: try start-process via cmd on windows
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/C", "start", "", target)
		if err := cmd.Start(); err != nil {
			return "launch error: " + err.Error()
		}
		return fmt.Sprintf("Launching %s...", target)
	}

	//fallback for linux: try exec.Command
	cmd := exec.Command(target)
	if err := cmd.Start(); err != nil {
		//try splitting
		parts := strings.Fields(target)
		cmd = exec.Command(parts[0], parts[1:]...)
		if err := cmd.Start(); err != nil {
			return "launch error: " + err.Error()
		}
	}
	return fmt.Sprintf("Launching %s...", target)
}

// opens a file or URL with the default app
func CmdOpen(args []string) string {
	if len(args) == 0 {
		return "open: expected a file or url"
	}
	target := strings.Join(args, " ")
	if err := open.Run(target); err != nil {
		return "open error: " + err.Error()
	}
	return fmt.Sprintf("Opened %s", target)
}

// performs a recursive search for filenames containing pattern.
// we'll limit results to first 200 entries to avoid massive output.
func CmdFind(args []string) string {
	if len(args) == 0 {
		return "find: expected search pattern, e.g. find resume"
	}
	pattern := strings.ToLower(strings.Join(args, " "))
	root := "."
	if runtime.GOOS == "windows" {
		//search C: by default; if user provided absolute path use that
		root = "C:\\"
	}

	results := []string{}
	limit := 200
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			//skip permission errors
			return nil
		}
		//only check file or dir name
		name := strings.ToLower(d.Name())
		if strings.Contains(name, pattern) {
			results = append(results, path)
			if len(results) >= limit {
				return filepath.SkipDir
			}
		}
		//protect from insanely long names
		if utf8.RuneCountInString(path) > 200 {
			return nil
		}
		return nil
	})
	if len(results) == 0 {
		return "No results found."
	}
	return strings.Join(results, "\n")
}

// returns a small system status summary.
func CmdSysStatus() string {
	var sb strings.Builder
	sb.WriteString("System status:\n")
	sb.WriteString(fmt.Sprintf("  OS: %s/%s\n", runtime.GOOS, runtime.GOARCH))
	sb.WriteString(fmt.Sprintf("  CPUs: %d\n", runtime.NumCPU()))
	//memory stats via runtime
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	sb.WriteString(fmt.Sprintf("  Alloc: %d KB\n", m.Alloc/1024))
	sb.WriteString(fmt.Sprintf("  Sys: %d KB\n", m.Sys/1024))
	sb.WriteString(fmt.Sprintf("  Goroutines: %d\n", runtime.NumGoroutine()))
	sb.WriteString(fmt.Sprintf("  Time: %s\n", time.Now().Format(time.RFC1123)))
	return sb.String()
}
