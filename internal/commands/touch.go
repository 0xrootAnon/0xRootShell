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
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func CmdTouch(args []string) string {
	if len(args) == 0 {
		return "touch: usage: touch [options] <file>...\nOptions: -p|--parents, -c|--no-create, -t <timestamp>, -r <ref>, -a, -m"
	}

	createParents := false
	noCreate := false
	setTime := time.Time{}
	hasSetTime := false
	refFile := ""
	onlyA := false
	onlyM := false
	remaining := []string{}

	i := 0
	for i < len(args) {
		a := args[i]
		if a == "-p" || a == "--parents" {
			createParents = true
			i++
			continue
		}
		if a == "-c" || a == "--no-create" {
			noCreate = true
			i++
			continue
		}
		if a == "-a" {
			onlyA = true
			i++
			continue
		}
		if a == "-m" {
			onlyM = true
			i++
			continue
		}
		if a == "-r" && i+1 < len(args) {
			refFile = args[i+1]
			i += 2
			continue
		}
		if a == "-t" && i+1 < len(args) {
			ts := args[i+1]
			parsed, perr := parseTimestamp(ts)
			if perr != nil {
				return "touch: invalid timestamp: " + perr.Error()
			}
			setTime = parsed
			hasSetTime = true
			i += 2
			continue
		}
		if strings.HasPrefix(a, "--time=") || strings.HasPrefix(a, "--timestamp=") {
			parts := strings.SplitN(a, "=", 2)
			if len(parts) == 2 {
				parsed, perr := parseTimestamp(parts[1])
				if perr != nil {
					return "touch: invalid timestamp: " + perr.Error()
				}
				setTime = parsed
				hasSetTime = true
				i++
				continue
			}
		}
		remaining = append(remaining, a)
		i++
	}

	if len(remaining) == 0 {
		return "touch: no files specified"
	}

	if refFile != "" {
		rp := expandPath(refFile)
		if !filepath.IsAbs(rp) {
			if wd, err := os.Getwd(); err == nil {
				rp = filepath.Join(wd, rp)
			}
		}
		fi, err := os.Stat(rp)
		if err != nil {
			return "touch: reference file error: " + err.Error()
		}
		setTime = fi.ModTime()
		hasSetTime = true
	}

	outLines := []string{}
	now := time.Now()

	for _, f := range remaining {
		p := expandPath(f)
		if !filepath.IsAbs(p) {
			if wd, err := os.Getwd(); err == nil {
				p = filepath.Join(wd, p)
			}
		}

		dir := filepath.Dir(p)
		if createParents {
			if err := os.MkdirAll(dir, 0755); err != nil {
				outLines = append(outLines, fmt.Sprintf("%s: failed to create parent dirs: %v", f, err))
				continue
			}
		} else {
			if dir != "." {
				if _, err := os.Stat(dir); err != nil {
					if os.IsNotExist(err) {
						if noCreate {
							outLines = append(outLines, fmt.Sprintf("%s: parent directory does not exist", f))
							continue
						}
						if err := os.MkdirAll(dir, 0755); err != nil {
							outLines = append(outLines, fmt.Sprintf("%s: failed to create parent dirs: %v", f, err))
							continue
						}
					}
				}
			}
		}

		fi, err := os.Stat(p)
		if err != nil {
			if os.IsNotExist(err) {
				if noCreate {
					outLines = append(outLines, fmt.Sprintf("%s: does not exist (not created due to --no-create)", f))
					continue
				}
				fd, ferr := os.OpenFile(p, os.O_CREATE|os.O_WRONLY, 0644)
				if ferr != nil {
					outLines = append(outLines, fmt.Sprintf("%s: create failed: %v", f, ferr))
					continue
				}
				_ = fd.Close()
				if hasSetTime {
					if err := setFileTimes(p, setTime, onlyA, onlyM); err != nil {
						outLines = append(outLines, fmt.Sprintf("%s: created but time set failed: %v", f, err))
						continue
					}
				} else {
					if err := setFileTimes(p, now, onlyA, onlyM); err != nil {
						outLines = append(outLines, fmt.Sprintf("%s: created but time set failed: %v", f, err))
						continue
					}
				}
				outLines = append(outLines, fmt.Sprintf("Created %s", p))
				continue
			}
			outLines = append(outLines, fmt.Sprintf("%s: stat error: %v", f, err))
			continue
		}

		targetTime := now
		if hasSetTime {
			targetTime = setTime
		}
		if err := setFileTimes(p, targetTime, onlyA, onlyM); err != nil {
			outLines = append(outLines, fmt.Sprintf("%s: update time failed: %v", f, err))
			continue
		}
		if fi.IsDir() {
			outLines = append(outLines, fmt.Sprintf("Updated directory timestamp %s", p))
		} else {
			outLines = append(outLines, fmt.Sprintf("Touched %s", p))
		}
	}

	return strings.Join(outLines, "\n")
}

func parseTimestamp(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02_15:04:05",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02",
	}
	for _, L := range layouts {
		if t, err := time.Parse(L, s); err == nil {
			return t, nil
		}
	}
	if t, err := time.Parse("200601021504.05", s); err == nil {
		return t, nil
	}
	if t, err := time.Parse("200601021504", s); err == nil {
		return t, nil
	}
	if sec, err := strconvParseInt(s); err == nil {
		return time.Unix(sec, 0), nil
	}
	return time.Time{}, fmt.Errorf("unrecognized timestamp: %s", s)
}

func strconvParseInt(s string) (int64, error) {
	s = strings.TrimSpace(s)
	return strconv.ParseInt(s, 10, 64)
}

func setFileTimes(path string, t time.Time, onlyA, onlyM bool) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	atime := t
	mtime := t
	if onlyA && !onlyM {
		mtime = info.ModTime()
	}
	if onlyM && !onlyA {
		atime = info.ModTime()
	}
	err = os.Chtimes(path, atime, mtime)
	return err
}
