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
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const extCmdTimeout = 6 * time.Second

func runCommand(cmdName string, args []string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, cmdName, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return out.String(), fmt.Errorf("timed out after %s", timeout)
		}
		return out.String(), err
	}
	return out.String(), nil
}

func expand(p string) string {
	return expandPath(p)
}

func CmdMkdir(args []string) string {
	if len(args) == 0 {
		return "mkdir: usage: mkdir <dir> [<dir> ...]"
	}
	out := []string{}
	for _, a := range args {
		p := expand(a)
		if !filepath.IsAbs(p) {
			if wd, err := os.Getwd(); err == nil {
				p = filepath.Join(wd, p)
			}
		}
		if err := os.MkdirAll(p, 0755); err != nil {
			out = append(out, fmt.Sprintf("mkdir: %s: %v", a, err))
		} else {
			out = append(out, fmt.Sprintf("Created %s", p))
		}
	}
	return strings.Join(out, "\n")
}

func CmdRmdir(args []string) string {
	if len(args) == 0 {
		return "rmdir: usage: rmdir <dir> [--force|-r]"
	}
	force := false
	paths := []string{}
	for _, a := range args {
		if a == "-r" || a == "--force" {
			force = true
			continue
		}
		paths = append(paths, a)
	}
	out := []string{}
	for _, a := range paths {
		p := expand(a)
		if !filepath.IsAbs(p) {
			if wd, err := os.Getwd(); err == nil {
				p = filepath.Join(wd, p)
			}
		}
		info, err := os.Stat(p)
		if err != nil {
			out = append(out, fmt.Sprintf("rmdir: %s: %v", a, err))
			continue
		}
		if !info.IsDir() {
			out = append(out, fmt.Sprintf("rmdir: %s: not a directory", a))
			continue
		}
		if force {
			if err := os.RemoveAll(p); err != nil {
				out = append(out, fmt.Sprintf("rmdir: %s: %v", a, err))
			} else {
				out = append(out, fmt.Sprintf("Removed (recursively) %s", p))
			}
			continue
		}
		f, err := os.Open(p)
		if err != nil {
			out = append(out, fmt.Sprintf("rmdir: %s: %v", a, err))
			continue
		}
		names, _ := f.Readdirnames(1)
		f.Close()
		if len(names) > 0 {
			out = append(out, fmt.Sprintf("rmdir: %s: directory not empty (use -r to remove)", a))
			continue
		}
		if err := os.Remove(p); err != nil {
			out = append(out, fmt.Sprintf("rmdir: %s: %v", a, err))
		} else {
			out = append(out, fmt.Sprintf("Removed %s", p))
		}
	}
	return strings.Join(out, "\n")
}

func deleteTargets(targets []string, recursive bool) (string, error) {
	if len(targets) == 0 {
		return "", errors.New("del/rm: expected target(s)")
	}
	out := []string{}
	for _, t := range targets {
		if strings.ContainsAny(t, "*?[]") {
			matches, _ := filepath.Glob(t)
			if len(matches) == 0 {
				out = append(out, fmt.Sprintf("No match: %s", t))
				continue
			}
			for _, m := range matches {
				if fi, err := os.Stat(m); err == nil {
					if fi.IsDir() {
						if recursive {
							if err := os.RemoveAll(m); err != nil {
								out = append(out, fmt.Sprintf("Failed to remove dir %s: %v", m, err))
							} else {
								out = append(out, fmt.Sprintf("Removed dir %s", m))
							}
						} else {
							out = append(out, fmt.Sprintf("Skipping dir %s (use -r to remove)", m))
						}
					} else {
						if err := os.Remove(m); err != nil {
							out = append(out, fmt.Sprintf("Failed to remove %s: %v", m, err))
						} else {
							out = append(out, fmt.Sprintf("Deleted %s", m))
						}
					}
				} else {
					out = append(out, fmt.Sprintf("Missing: %s", m))
				}
			}
			continue
		}

		p := expand(t)
		if !filepath.IsAbs(p) {
			if wd, err := os.Getwd(); err == nil {
				p = filepath.Join(wd, p)
			}
		}
		fi, err := os.Stat(p)
		if err != nil {
			out = append(out, fmt.Sprintf("Missing: %s", t))
			continue
		}
		if fi.IsDir() {
			if recursive {
				if err := os.RemoveAll(p); err != nil {
					out = append(out, fmt.Sprintf("Failed to remove dir %s: %v", p, err))
				} else {
					out = append(out, fmt.Sprintf("Removed dir %s", p))
				}
			} else {
				out = append(out, fmt.Sprintf("Skipping dir %s (use -r to remove)", p))
			}
		} else {
			if err := os.Remove(p); err != nil {
				out = append(out, fmt.Sprintf("Failed to delete %s: %v", p, err))
			} else {
				out = append(out, fmt.Sprintf("Deleted %s", p))
			}
		}
	}
	return strings.Join(out, "\n"), nil
}

func CmdDel(args []string) string {
	if len(args) == 0 {
		return "del: usage: del [-r|--recursive] <target> [<target> ...]"
	}
	recursive := false
	targets := []string{}
	for _, a := range args {
		if a == "-r" || a == "--recursive" {
			recursive = true
			continue
		}
		targets = append(targets, a)
	}
	s, err := deleteTargets(targets, recursive)
	if err != nil {
		return "del: " + err.Error() + "\n" + s
	}
	return s
}
func CmdRm(args []string) string {
	// rm is alias to del
	return CmdDel(args)
}

func CmdCp(args []string) string {
	if len(args) < 2 {
		return "cp: usage: cp <src> <dst>  OR cp <src1> <src2> ... <dstDir>"
	}
	dst := expand(args[len(args)-1])
	if !filepath.IsAbs(dst) {
		if wd, err := os.Getwd(); err == nil {
			dst = filepath.Join(wd, dst)
		}
	}
	srcs := args[:len(args)-1]
	if len(srcs) > 1 {
		if info, err := os.Stat(dst); err != nil || !info.IsDir() {
			return "cp: when copying multiple sources, destination must be an existing directory"
		}
	}
	out := []string{}
	for _, s := range srcs {
		sp := expand(s)
		if !filepath.IsAbs(sp) {
			if wd, err := os.Getwd(); err == nil {
				sp = filepath.Join(wd, sp)
			}
		}
		info, err := os.Stat(sp)
		if err != nil {
			out = append(out, fmt.Sprintf("cp: %s: %v", s, err))
			continue
		}
		target := dst
		if len(srcs) > 1 || (info.IsDir() && (info.IsDir())) {
			target = filepath.Join(dst, filepath.Base(sp))
		}
		if err := copyFileOrDir(sp, target); err != nil {
			out = append(out, fmt.Sprintf("cp: failed %s -> %s: %v", sp, target, err))
		} else {
			out = append(out, fmt.Sprintf("Copied %s -> %s", sp, target))
		}
	}
	return strings.Join(out, "\n")
}

func CmdMv(args []string) string {
	if len(args) < 2 {
		return "mv: usage: mv <src> <dst>"
	}
	src := expand(args[0])
	if !filepath.IsAbs(src) {
		if wd, err := os.Getwd(); err == nil {
			src = filepath.Join(wd, src)
		}
	}
	dst := expand(args[1])
	if !filepath.IsAbs(dst) {
		if wd, err := os.Getwd(); err == nil {
			dst = filepath.Join(wd, dst)
		}
	}
	if err := os.Rename(src, dst); err == nil {
		return fmt.Sprintf("Moved %s -> %s", src, dst)
	}
	if err := copyFileOrDir(src, dst); err == nil {
		_ = os.RemoveAll(src)
		return fmt.Sprintf("Moved %s -> %s", src, dst)
	}
	return fmt.Sprintf("mv: failed to move %s -> %s", src, dst)
}

func CmdCat(args []string) string {
	if len(args) == 0 {
		return "cat: usage: cat <file> [file2 ...]"
	}
	out := &strings.Builder{}
	for i, a := range args {
		p := expand(a)
		if !filepath.IsAbs(p) {
			if wd, err := os.Getwd(); err == nil {
				p = filepath.Join(wd, p)
			}
		}
		f, err := os.Open(p)
		if err != nil {
			return fmt.Sprintf("cat: %s: %v", a, err)
		}
		if len(args) > 1 {
			out.WriteString(fmt.Sprintf("=== %s ===\n", a))
		}
		_, err = io.Copy(out, f)
		f.Close()
		if err != nil {
			return fmt.Sprintf("cat: read error %s: %v", a, err)
		}
		if i < len(args)-1 {
			out.WriteString("\n")
		}
	}
	return out.String()
}

func CmdGrep(args []string) string {
	if len(args) < 2 {
		return "grep: usage: grep [-i] [-n] <pattern> <file> [file...]"
	}
	ignoreCase := false
	showNumber := false
	toks := []string{}
	for _, t := range args {
		if t == "-i" {
			ignoreCase = true
			continue
		}
		if t == "-n" {
			showNumber = true
			continue
		}
		toks = append(toks, t)
	}
	if len(toks) < 2 {
		return "grep: usage: grep [-i] [-n] <pattern> <file> [file...]"
	}
	pattern := toks[0]
	files := toks[1:]
	out := &strings.Builder{}
	for _, f := range files {
		p := expand(f)
		if !filepath.IsAbs(p) {
			if wd, err := os.Getwd(); err == nil {
				p = filepath.Join(wd, p)
			}
		}
		file, err := os.Open(p)
		if err != nil {
			out.WriteString(fmt.Sprintf("grep: %s: %v\n", f, err))
			continue
		}
		sc := bufio.NewScanner(file)
		ln := 0
		for sc.Scan() {
			ln++
			line := sc.Text()
			hay := line
			pat := pattern
			if ignoreCase {
				hay = strings.ToLower(hay)
				pat = strings.ToLower(pat)
			}
			if strings.Contains(hay, pat) {
				if showNumber {
					out.WriteString(fmt.Sprintf("%s:%d: %s\n", f, ln, line))
				} else {
					out.WriteString(fmt.Sprintf("%s: %s\n", f, line))
				}
			}
		}
		file.Close()
	}
	res := strings.TrimSpace(out.String())
	if res == "" {
		return "grep: no matches"
	}
	return res
}

func CmdTasklist(args []string) string {
	if isWindows() {
		out, err := runCommand("tasklist", []string{"/FO", "TABLE"}, extCmdTimeout)
		if err != nil {
			out2, err2 := runCommand("tasklist", []string{}, extCmdTimeout)
			if err2 != nil {
				return "tasklist: error: " + err.Error() + " | " + err2.Error()
			}
			return out2
		}
		return strings.TrimSpace(out)
	}
	out, err := runCommand("ps", []string{"-eo", "pid,comm,%cpu,%mem", "--sort=-%cpu"}, extCmdTimeout)
	if err != nil {
		return "tasklist: error: " + err.Error()
	}
	return strings.TrimSpace(out)
}

func CmdTaskkill(args []string) string {
	if len(args) == 0 {
		return "taskkill: usage: taskkill <pid> | taskkill /IM <name> | taskkill <name>"
	}
	if isWindows() {
		if args[0] == "/IM" && len(args) > 1 {
			name := args[1]
			out, err := runCommand("taskkill", []string{"/IM", name, "/F"}, extCmdTimeout)
			if err != nil {
				return "taskkill: " + err.Error() + " | " + out
			}
			return strings.TrimSpace(out)
		}
		if pid, err := strconv.Atoi(args[0]); err == nil {
			out, err := runCommand("taskkill", []string{"/PID", fmt.Sprintf("%d", pid), "/F"}, extCmdTimeout)
			if err != nil {
				return "taskkill: " + err.Error() + " | " + out
			}
			return strings.TrimSpace(out)
		}
		out, err := runCommand("taskkill", []string{"/IM", args[0], "/F"}, extCmdTimeout)
		if err != nil {
			return "taskkill: " + err.Error() + " | " + out
		}
		return strings.TrimSpace(out)
	}
	if pid, err := strconv.Atoi(args[0]); err == nil {
		out, err := runCommand("kill", []string{"-9", fmt.Sprintf("%d", pid)}, extCmdTimeout)
		if err != nil {
			return "taskkill: " + err.Error() + " | " + out
		}
		return fmt.Sprintf("killed %d", pid)
	}
	out, err := runCommand("pkill", []string{"-f", args[0]}, extCmdTimeout)
	if err != nil {
		// pkill returns non-zero if no process matched; return message
		return "taskkill: " + err.Error() + " | " + out
	}
	return strings.TrimSpace(out)
}

func CmdGetVolume(args []string) string {
	if isWindows() {
		out, err := runCommand("wmic", []string{"logicaldisk", "get", "Caption,FreeSpace,Size,VolumeName"}, extCmdTimeout)
		if err == nil && strings.TrimSpace(out) != "" {
			return strings.TrimSpace(out)
		}
		pout, err2 := runCommand("powershell", []string{"-NoProfile", "-Command", "Get-Volume | Select DriveLetter, SizeRemaining, Size | Format-Table -AutoSize"}, extCmdTimeout)
		if err2 == nil && strings.TrimSpace(pout) != "" {
			return strings.TrimSpace(pout)
		}
		if err != nil {
			return "get-volume: wmic error: " + err.Error()
		}
		return "get-volume: no output"
	}
	o, e := runCommand("df", []string{"-h"}, extCmdTimeout)
	if e != nil {
		return "get-volume: df error: " + e.Error()
	}
	return strings.TrimSpace(o)
}

func isWindows() bool {
	return strings.EqualFold(os.Getenv("OS"), "Windows_NT") || filepath.Separator == '\\'
}

func expandPath(p string) string {
	if p == "" {
		return p
	}

	p = os.ExpandEnv(p)

	if strings.HasPrefix(p, "~") {
		if p == "~" {
			if h, err := os.UserHomeDir(); err == nil && h != "" {
				p = h
			}
		} else if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
			if h, err := os.UserHomeDir(); err == nil && h != "" {
				p = filepath.Join(h, p[2:])
			}
		} else {
			//rare case: "~user/path", we don't attempt to resolve other users cross-platform,
			//so leaving it as-is (safer than guessing).
		}
	}

	if !filepath.IsAbs(p) {
		if wd, err := os.Getwd(); err == nil {
			p = filepath.Join(wd, p)
		}
	}

	p = filepath.Clean(p)

	if runtime.GOOS == "windows" {
		p = filepath.FromSlash(p)
	}

	return p
}

func copyFileOrDir(src, dst string) error {
	if src == "" || dst == "" {
		return errors.New("copy: src and dst must be non-empty")
	}

	src = expandPath(src)
	dst = expandPath(dst)

	srcInfo, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("copy: stat src: %w", err)
	}

	ensureParent := func(path string, mode os.FileMode) error {
		dir := filepath.Dir(path)
		if dir == "" || dir == "." {
			return nil
		}
		return os.MkdirAll(dir, mode|0755)
	}

	if srcInfo.Mode()&os.ModeSymlink != 0 {
		linkTarget, err := os.Readlink(src)
		if err != nil {
			return fmt.Errorf("copy: readlink src: %w", err)
		}

		if dInfo, derr := os.Stat(dst); derr == nil && dInfo.IsDir() {
			dst = filepath.Join(dst, filepath.Base(src))
		} else {
			_ = ensureParent(dst, 0755)
		}

		if err := os.Symlink(linkTarget, dst); err != nil {
			resolved := linkTarget
			if !filepath.IsAbs(resolved) {
				resolved = filepath.Join(filepath.Dir(src), linkTarget)
			}
			if fi, e := os.Stat(resolved); e == nil {
				if fi.IsDir() {
					return copyDirRecursive(resolved, dst)
				}
				return copyFileContents(resolved, dst, fi.Mode())
			}
			return os.WriteFile(dst, []byte("SYMLINK->"+linkTarget), 0644)
		}
		return nil
	}

	if srcInfo.IsDir() {
		if dInfo, derr := os.Stat(dst); derr == nil && dInfo.IsDir() {
			dst = filepath.Join(dst, filepath.Base(src))
		}
		return copyDirRecursive(src, dst)
	}

	if dInfo, derr := os.Stat(dst); derr == nil && dInfo.IsDir() {
		dst = filepath.Join(dst, filepath.Base(src))
	} else {
		if err := ensureParent(dst, srcInfo.Mode()); err != nil {
			return fmt.Errorf("copy: ensure parent: %w", err)
		}
	}

	return copyFileContents(src, dst, srcInfo.Mode())
}

func copyFileContents(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("copy: open src: %w", err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		if mkerr := os.MkdirAll(filepath.Dir(dst), 0755); mkerr == nil {
			out, err = os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
		}
		if err != nil {
			return fmt.Errorf("copy: create dst: %w", err)
		}
	}
	defer func() {
		_ = out.Sync()
		_ = out.Close()
	}()

	buf := make([]byte, 32*1024)
	if _, err := io.CopyBuffer(out, in, buf); err != nil {
		return fmt.Errorf("copy: copy data: %w", err)
	}

	if err := out.Chmod(mode.Perm()); err != nil {
		// non-fatal on some platforms
		_ = err
	}
	if fi, err := os.Stat(src); err == nil {
		_ = os.Chtimes(dst, fi.ModTime(), fi.ModTime())
	}
	return nil
}

func copyDirRecursive(src, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("copy: stat src dir: %w", err)
	}
	if !si.IsDir() {
		return fmt.Errorf("copy: source is not a directory")
	}

	if err := os.MkdirAll(dst, si.Mode().Perm()); err != nil {
		return fmt.Errorf("copy: mkdir dst: %w", err)
	}
	return filepath.WalkDir(src, func(path string, de os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)

		info, err := de.Info()
		if err != nil {
			return err
		}

		if info.IsDir() {
			if err := os.MkdirAll(target, info.Mode().Perm()); err != nil {
				return err
			}
			return os.Chtimes(target, info.ModTime(), info.ModTime())
		}

		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := os.Symlink(linkTarget, target); err != nil {
				resolved := linkTarget
				if !filepath.IsAbs(resolved) {
					resolved = filepath.Join(filepath.Dir(path), linkTarget)
				}
				if fi, e := os.Stat(resolved); e == nil {
					if fi.IsDir() {
						if err := copyDirRecursive(resolved, target); err != nil {
							return err
						}
						return nil
					}
					if err := copyFileContents(resolved, target, fi.Mode()); err != nil {
						return err
					}
					return nil
				}
				return os.WriteFile(target, []byte("SYMLINK->"+linkTarget), 0644)
			}
			return nil
		}

		if err := copyFileContents(path, target, info.Mode()); err != nil {
			return err
		}
		return nil
	})
}
