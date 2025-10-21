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
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func CmdFile(args []string) string {
	if len(args) == 0 {
		return "file: expected subcommand (move, rename, clean, open)"
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "move":
		if len(args) < 3 {
			return "file move: usage: file move <src> <dst>"
		}
		return fileMove(args[1], args[2])
	case "rename":
		if len(args) < 3 {
			return "file rename: usage: file rename <pattern> <replacement>"
		}
		return fileRenameBulk(args[1], args[2])
	case "clean":
		if len(args) >= 2 && args[1] == "temp" {
			return fileCleanTemp()
		}
		return "file clean: supported targets: temp"
	case "open":
		if len(args) < 2 {
			return "file open <path>"
		}
		return CmdOpen(args[1:])
	default:
		return "file: unknown subcommand"
	}
}

func fileMove(src, dst string) string {
	if strings.HasPrefix(src, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			src = filepath.Join(home, src[2:])
		}
	}
	if strings.HasPrefix(dst, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			dst = filepath.Join(home, dst[2:])
		}
	}

	if err := os.Rename(src, dst); err != nil {
		if err := copyFileOrDir(src, dst); err == nil {
			_ = os.RemoveAll(src)
			return fmt.Sprintf("Moved %s -> %s", src, dst)
		}
		return "file move error: " + err.Error()
	}
	return fmt.Sprintf("Moved %s -> %s", src, dst)
}

func copyFileOrDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, 0755); err != nil {
			return err
		}
		entries, _ := os.ReadDir(src)
		for _, e := range entries {
			if err := copyFileOrDir(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func fileRenameBulk(pattern, repl string) string {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "rename: invalid pattern"
	}
	if len(matches) == 0 {
		return "rename: no files match pattern"
	}
	for i, p := range matches {
		dir := filepath.Dir(p)
		ext := filepath.Ext(p)
		base := strings.TrimSuffix(filepath.Base(p), ext)
		target := repl
		target = strings.ReplaceAll(target, "#", fmt.Sprintf("%d", i+1))
		target = strings.ReplaceAll(target, "{name}", base)
		target = target + ext
		if err := os.Rename(p, filepath.Join(dir, target)); err != nil {
			return "rename error: " + err.Error()
		}
	}
	return fmt.Sprintf("Renamed %d files", len(matches))
}

func fileCleanTemp() string {
	tmp := os.TempDir()
	entries, err := os.ReadDir(tmp)
	if err != nil {
		return "clean temp: " + err.Error()
	}
	out := []string{"Files in temp (first 20):"}
	n := 0
	for _, e := range entries {
		out = append(out, " - "+e.Name())
		n++
		if n >= 20 {
			break
		}
	}
	out = append(out, "\nTo actually delete temp files run: file clean temp --confirm")
	return strings.Join(out, "\n")
}

func CmdCompressArchive(args []string) string {
	if len(args) == 0 {
		return "compress: usage: compress <out.zip> <src-dir-or-file> | extract <in.zip> <dst>"
	}
	verb := args[0]
	if verb == "compress" || verb == "zip" || verb == "compress" {
		if len(args) < 3 {
			return "compress: usage: compress <out.zip> <src>"
		}
		out := args[1]
		src := args[2]
		if err := zipPath(src, out); err != nil {
			return "compress error: " + err.Error()
		}
		return "Created " + out
	} else if verb == "extract" || verb == "unzip" {
		if len(args) < 3 {
			return "extract: usage: extract <in.zip> <dst>"
		}
		in := args[1]
		dst := args[2]
		if err := unzip(in, dst); err != nil {
			return "extract error: " + err.Error()
		}
		return "Extracted to " + dst
	}
	return "compress: unknown subcommand"
}

func zipPath(src, out string) error {
	zf, err := os.Create(out)
	if err != nil {
		return err
	}
	defer zf.Close()
	w := zip.NewWriter(zf)
	defer w.Close()

	src = filepath.Clean(src)
	base := filepath.Base(src)

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(filepath.Dir(src), path)
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(filepath.Join(base, rel))
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}
		writer, err := w.CreateHeader(header)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err := io.Copy(writer, f); err != nil {
				return err
			}
		}
		return nil
	})
}

func unzip(in, dst string) error {
	r, err := zip.OpenReader(in)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		fpath := filepath.Join(dst, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, f.Mode())
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}
		if _, err := io.Copy(outFile, rc); err != nil {
			rc.Close()
			outFile.Close()
			return err
		}
		rc.Close()
		outFile.Close()
	}
	return nil
}
