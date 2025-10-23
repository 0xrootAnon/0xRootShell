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
	"strings"
	"time"
)

func killBrowserProcesses(names []string) {
	for _, n := range names {
		exec.Command("taskkill", "/F", "/IM", n).Run()
	}
	time.Sleep(400 * time.Millisecond)
}

func tryRemove(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	if err := os.Remove(path); err == nil {
		return nil
	} else {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		f.Close()
		return nil
	}
}

func expandVars(p string) string {
	return os.ExpandEnv(p)
}

func gatherChromeLikeHistoryPaths(base string) []string {
	out := []string{}
	base = expandVars(base)
	if fi, err := os.Stat(base); err == nil && fi.IsDir() {
		entries, _ := os.ReadDir(base)
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			profile := filepath.Join(base, e.Name())
			candidates := []string{
				filepath.Join(profile, "History"),
				filepath.Join(profile, "Top Sites"),
				filepath.Join(profile, "History-journal"),
				filepath.Join(profile, "Network Action Predictor"),
			}
			for _, c := range candidates {
				out = append(out, c)
			}
		}
	}
	return out
}

func gatherFirefoxHistoryPaths(base string) []string {
	out := []string{}
	base = expandVars(base)
	if fi, err := os.Stat(base); err == nil && fi.IsDir() {
		entries, _ := os.ReadDir(base)
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			profile := filepath.Join(base, e.Name())
			candidates := []string{
				filepath.Join(profile, "places.sqlite"),
				filepath.Join(profile, "places.sqlite-shm"),
				filepath.Join(profile, "places.sqlite-wal"),
			}
			for _, c := range candidates {
				out = append(out, c)
			}
		}
	}
	return out
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func findExecutableCandidates(cands []string) string {
	for _, c := range cands {
		c = expandVars(c)
		if fileExists(c) {
			return c
		}
		if p, err := exec.LookPath(c); err == nil && p != "" {
			return p
		}
	}
	return ""
}

func removeFiles(paths []string) (deleted []string, failed []string) {
	for _, p := range paths {
		if p == "" {
			continue
		}
		if tryRemove(p) == nil {
			deleted = append(deleted, p)
			continue
		}
		if !fileExists(p) {
			continue
		}
		failed = append(failed, p)
	}
	return
}

func CmdClearBrowserHistory(args []string) string {
	var out []string
	browsersKilled := []string{"chrome.exe", "msedge.exe", "brave.exe", "vivaldi.exe", "opera.exe", "firefox.exe"}
	killBrowserProcesses(browsersKilled)
	out = append(out, "attempted to stop browser processes")

	local := os.Getenv("LOCALAPPDATA")
	app := os.Getenv("APPDATA")
	profileCandidates := []string{}
	profileCandidates = append(profileCandidates, gatherChromeLikeHistoryPaths(filepath.Join(local, "Google", "Chrome", "User Data"))...)
	profileCandidates = append(profileCandidates, gatherChromeLikeHistoryPaths(filepath.Join(local, "Microsoft", "Edge", "User Data"))...)
	profileCandidates = append(profileCandidates, gatherChromeLikeHistoryPaths(filepath.Join(local, "BraveSoftware", "Brave-Browser", "User Data"))...)
	profileCandidates = append(profileCandidates, gatherChromeLikeHistoryPaths(filepath.Join(local, "Vivaldi", "User Data"))...)
	profileCandidates = append(profileCandidates, gatherChromeLikeHistoryPaths(filepath.Join(app, "Opera Software", "Opera Stable"))...)
	firefoxPaths := gatherFirefoxHistoryPaths(filepath.Join(app, "Mozilla", "Firefox", "Profiles"))
	profileCandidates = append(profileCandidates, firefoxPaths...)
	deleted, failed := removeFiles(profileCandidates)
	if len(deleted) > 0 {
		out = append(out, "removed:")
		for _, d := range deleted {
			out = append(out, "  "+d)
		}
	} else {
		out = append(out, "no history files removed")
	}
	if len(failed) > 0 {
		out = append(out, "failed to remove:")
		for _, f := range failed {
			out = append(out, "  "+f)
		}
	}
	if len(deleted) == 0 && len(failed) == 0 {
		out = append(out, "no supported browser history files found")
	}
	return strings.Join(out, "\n")
}

func CmdBrowsePrivate(args []string) string {
	url := ""
	if len(args) > 0 {
		url = args[0]
	}
	programs := []struct {
		name  string
		args  func(string) []string
		cands []string
	}{
		{
			name: "chrome",
			args: func(u string) []string {
				if u == "" {
					return []string{"--incognito"}
				}
				return []string{"--incognito", u}
			},
			cands: []string{
				filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
				filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
				"chrome.exe",
			},
		},
		{
			name: "edge",
			args: func(u string) []string {
				if u == "" {
					return []string{"--inprivate"}
				}
				return []string{"--inprivate", u}
			},
			cands: []string{
				filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
				filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
				"msedge.exe",
			},
		},
		{
			name: "brave",
			args: func(u string) []string {
				if u == "" {
					return []string{"--incognito"}
				}
				return []string{"--incognito", u}
			},
			cands: []string{
				filepath.Join(os.Getenv("ProgramFiles"), "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
				filepath.Join(os.Getenv("ProgramFiles(x86)"), "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
				"brave.exe",
			},
		},
		{
			name: "firefox",
			args: func(u string) []string {
				if u == "" {
					return []string{"-private-window"}
				}
				return []string{"-private-window", u}
			},
			cands: []string{
				filepath.Join(os.Getenv("ProgramFiles"), "Mozilla Firefox", "firefox.exe"),
				filepath.Join(os.Getenv("ProgramFiles(x86)"), "Mozilla Firefox", "firefox.exe"),
				"firefox.exe",
			},
		},
		{
			name: "vivaldi",
			args: func(u string) []string {
				if u == "" {
					return []string{"--incognito"}
				}
				return []string{"--incognito", u}
			},
			cands: []string{
				filepath.Join(os.Getenv("ProgramFiles"), "Vivaldi", "Application", "vivaldi.exe"),
				filepath.Join(os.Getenv("ProgramFiles(x86)"), "Vivaldi", "Application", "vivaldi.exe"),
				"vivaldi.exe",
			},
		},
		{
			name: "opera",
			args: func(u string) []string {
				if u == "" {
					return []string{"--private"}
				}
				return []string{"--private", u}
			},
			cands: []string{
				filepath.Join(os.Getenv("ProgramFiles"), "Opera", "launcher.exe"),
				filepath.Join(os.Getenv("ProgramFiles(x86)"), "Opera", "launcher.exe"),
				"opera.exe",
				"launcher.exe",
			},
		},
	}

	for _, p := range programs {
		exe := findExecutableCandidates(p.cands)
		if exe == "" {
			continue
		}
		cmd := exec.Command(exe, p.args(url)...)
		if err := cmd.Start(); err == nil {
			if url == "" {
				return fmt.Sprintf("Launched %s in private mode", p.name)
			}
			return fmt.Sprintf("Launched %s in private mode with %s", p.name, url)
		}
	}

	if url == "" {
		if err := exec.Command("cmd", "/C", "start", "msedge", "--inprivate").Start(); err == nil {
			return "Launched default browser in private mode"
		}
		return "Could not launch a private browser window"
	}

	if err := exec.Command("cmd", "/C", "start", url).Start(); err == nil {
		return "Opened URL in default browser (private mode not available)"
	}
	return "Could not open URL"
}
