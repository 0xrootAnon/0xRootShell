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
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

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
