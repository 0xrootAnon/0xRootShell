package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// CmdLS: quick directory listing
func CmdLS(args []string) string {
	dir := "."
	if len(args) > 0 && args[0] != "" {
		dir = args[0]
	}
	dir = expandPath(dir)
	if !filepath.IsAbs(dir) {
		if wd, err := os.Getwd(); err == nil {
			dir = filepath.Join(wd, dir)
		}
	}
	info, err := os.Stat(dir)
	if err != nil {
		return "ls: " + err.Error()
	}
	if !info.IsDir() {
		return dir
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "ls: " + err.Error()
	}
	type ent struct {
		Name string
		Size int64
		Time time.Time
		Dir  bool
	}
	out := []ent{}
	for _, e := range entries {
		fi, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, ent{Name: e.Name(), Size: fi.Size(), Time: fi.ModTime(), Dir: e.IsDir()})
	}
	sort.Slice(out, func(i, j int) bool {
		// directories first
		if out[i].Dir != out[j].Dir {
			return out[i].Dir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	var b strings.Builder
	for _, e := range out {
		if e.Dir {
			fmt.Fprintf(&b, "[DIR] %s\t%s\n", e.Name, e.Time.Format("2006-01-02 15:04"))
		} else {
			fmt.Fprintf(&b, "      %s\t%d bytes\t%s\n", e.Name, e.Size, e.Time.Format("2006-01-02 15:04"))
		}
	}
	return b.String()
}

// ensure safeExists already present in commands.go
func SafeWalk(root, pattern string, timeoutSecs int) ([]string, error) {
	col := []string{}
	start := time.Now()
	limit := time.Duration(timeoutSecs) * time.Second
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if time.Since(start) > limit {
			return filepath.SkipDir
		}
		if err != nil {
			return nil
		}
		if strings.Contains(strings.ToLower(d.Name()), strings.ToLower(pattern)) {
			col = append(col, path)
		}
		return nil
	})
	if len(col) == 0 {
		return nil, fmt.Errorf("no results")
	}
	return col, nil
}
