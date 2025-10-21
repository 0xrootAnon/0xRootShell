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
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/skratchdot/open-golang/open"
)

func expandPath(p string) string {
	if p == "" {
		return p
	}
	if p == "~" {
		if h, err := os.UserHomeDir(); err == nil {
			return h
		}
		return p
	}
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		if h, err := os.UserHomeDir(); err == nil {
			return filepath.Join(h, p[2:])
		}
	}
	return p
}

func looksLikeURL(s string) bool {
	if strings.Contains(s, "://") {
		return true
	}
	if strings.Contains(s, ".") && !strings.ContainsAny(s, `/\`) {
		return true
	}
	if u, err := url.Parse(s); err == nil && u.Scheme != "" && u.Host != "" {
		return true
	}
	return false
}

func prependHTTPSIfNeeded(s string) string {
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return s
	}
	return "https://" + s
}

func runOpen(target string) error {
	if err := open.Run(target); err == nil {
		return nil
	}

	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/C", "start", "", target)
		return cmd.Start()
	}
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("open", target)
		return cmd.Start()
	}
	cmd := exec.Command("xdg-open", target)
	return cmd.Start()
}

func safeExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func CmdLaunch(args []string) string {
	if len(args) == 0 {
		return "launch: expected an app name or URL, e.g. `launch chrome` or `launch https://example.com`"
	}
	target := strings.Join(args, " ")

	if looksLikeURL(target) {
		target = prependHTTPSIfNeeded(target)
		if err := runOpen(target); err != nil {
			return "launch error: " + err.Error()
		}
		return fmt.Sprintf("Launching %s...", target)
	}

	targetExpanded := expandPath(target)

	if strings.ContainsAny(targetExpanded, `/\`) || safeExists(targetExpanded) {
		if !filepath.IsAbs(targetExpanded) {
			if wd, err := os.Getwd(); err == nil {
				targetExpanded = filepath.Join(wd, targetExpanded)
			}
		}
		if safeExists(targetExpanded) {
			if err := runOpen(targetExpanded); err != nil {
				return "launch error: " + err.Error()
			}
			return fmt.Sprintf("Launching %s...", targetExpanded)
		}
	}

	parts := strings.Fields(target)
	cmd := exec.Command(parts[0], parts[1:]...)
	if err := cmd.Start(); err != nil {
		if err2 := runOpen(target); err2 == nil {
			return fmt.Sprintf("Launching %s...", target)
		}
		return "launch error: " + err.Error()
	}
	return fmt.Sprintf("Launching %s...", target)
}

func CmdOpen(args []string) string {
	if len(args) == 0 {
		return "open: expected a file or url, e.g. `open ~/Downloads` or `open reddit.com`"
	}
	target := strings.Join(args, " ")

	target = expandPath(target)

	if !strings.Contains(target, "://") {
		if !filepath.IsAbs(target) {
			if wd, err := os.Getwd(); err == nil {
				try := filepath.Join(wd, target)
				if safeExists(try) {
					target = try
				}
			}
		}
		if safeExists(target) {
			if err := runOpen(target); err != nil {
				return "open error: " + err.Error()
			}
			return fmt.Sprintf("Opened %s", target)
		}
	}

	if looksLikeURL(target) {
		target = prependHTTPSIfNeeded(target)
		if err := runOpen(target); err != nil {
			return "open error: " + err.Error()
		}
		return fmt.Sprintf("Opened %s", target)
	}

	if looksLikeURL(target) {
		target = prependHTTPSIfNeeded(target)
		if err := runOpen(target); err != nil {
			return "open error: " + err.Error()
		}
		return fmt.Sprintf("Opened %s", target)
	}

	target = expandPath(target)

	if !filepath.IsAbs(target) {
		if wd, err := os.Getwd(); err == nil {
			try := filepath.Join(wd, target)
			if safeExists(try) {
				target = try
			}
		}
	}

	if !safeExists(target) && !strings.ContainsAny(target, `/\`) {
		home, _ := os.UserHomeDir()
		aliases := map[string]string{
			"downloads": filepath.Join(home, "Downloads"),
			"desktop":   filepath.Join(home, "Desktop"),
			"documents": filepath.Join(home, "Documents"),
		}
		l := strings.ToLower(target)
		if p, ok := aliases[l]; ok && safeExists(p) {
			target = p
		}
	}

	if !safeExists(target) {
		if !strings.ContainsAny(target, `/\`) {
			return fmt.Sprintf("open: '%s' not found. Try `find %s` or provide a full/relative path.", target, target)
		}
		return "open error: target not found"
	}

	if err := runOpen(target); err != nil {
		return "open error: " + err.Error()
	}
	return fmt.Sprintf("Opened %s", target)
}

func CmdFind(args []string) string {
	if len(args) == 0 {
		return "find: expected search pattern, e.g. `find resume`"
	}

	all := false
	parts := []string{}
	for _, a := range args {
		if a == "--all" || a == "-a" {
			all = true
			continue
		}
		parts = append(parts, a)
	}
	pattern := strings.ToLower(strings.Join(parts, " "))
	if pattern == "" {
		return "find: empty pattern"
	}

	wd, _ := os.Getwd()
	if len(parts) > 0 && strings.ContainsAny(parts[0], `/\`) {
		candidate := expandPath(parts[0])
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(wd, candidate)
		}
		if safeExists(candidate) {
			return candidate
		}
	}

	if strings.Contains(pattern, ".") {
		quickResults := []string{}
		limit := 200
		start := time.Now()
		_ = filepath.WalkDir(wd, func(path string, d os.DirEntry, err error) error {
			if time.Since(start) > 3*time.Second {
				return filepath.SkipDir
			}
			if err != nil {
				return nil
			}
			name := strings.ToLower(d.Name())
			if strings.Contains(name, pattern) {
				quickResults = append(quickResults, path)
				if len(quickResults) >= limit {
					return filepath.SkipDir
				}
			}
			if utf8.RuneCountInString(path) > 400 {
				return nil
			}
			return nil
		})
		if len(quickResults) > 0 {
			if len(quickResults) > 100 {
				quickResults = quickResults[:100]
			}
			return strings.Join(quickResults, "\n")
		}
	}

	root := "."
	if all {
		if runtime.GOOS == "windows" {
			root = "C:\\"
		} else {
			root = "/"
		}
	} else {
		if h, err := os.UserHomeDir(); err == nil {
			root = h
		}
		if len(parts) > 0 && strings.ContainsAny(parts[0], `/\`) {
			root = expandPath(parts[0])
		}
	}

	timeout := 8 * time.Second
	if all {
		timeout = 20 * time.Second
	}

	results := []string{}
	limit := 500
	start := time.Now()
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if time.Since(start) > timeout {
			return filepath.SkipDir
		}
		if err != nil {
			return nil
		}
		name := strings.ToLower(d.Name())
		if strings.Contains(name, pattern) {
			results = append(results, path)
			if len(results) >= limit {
				return filepath.SkipDir
			}
		}
		if utf8.RuneCountInString(path) > 400 {
			return nil
		}
		return nil
	})

	if len(results) == 0 {
		if !all {
			return "No results found. Try: `find <pattern> --all` to search entire disk (may be slow)."
		}
		return "No results found."
	}

	if len(results) > 200 {
		results = results[:200]
	}
	return strings.Join(results, "\n")
}
