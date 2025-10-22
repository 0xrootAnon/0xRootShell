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
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func StartScan(args []string, ch chan string) {
	if runtime.GOOS != "windows" {
		ch <- "scan: Windows Defender supported only on Windows."
		return
	}

	scanType, mpScanType, ok := parseScanArgs(args)
	if !ok {
		ch <- "scan: unrecognized target. Use `scan system` or `scan system full`."
		return
	}

	ch <- fmt.Sprintf("Starting Windows Defender %s...", scanType)

	ctxTimeout := 15 * time.Minute
	if err := runDefenderPowershellScanStream(scanType, ch, ctxTimeout); err == nil {
		ch <- fmt.Sprintf("Windows Defender %s finished.", scanType)
		return
	} else {
		ch <- fmt.Sprintf("PowerShell scan failed: %v — falling back to MpCmdRun.exe", err)
	}

	if err := runMpCmdRunScanStream(mpScanType, ch, ctxTimeout); err == nil {
		ch <- fmt.Sprintf("Windows Defender (MpCmdRun) %s finished.", scanType)
		return
	} else {
		ch <- fmt.Sprintf("MpCmdRun scan failed: %v", err)
	}

	ch <- "scan: failed to complete. See messages above for details."
}

func CmdScan(args []string) string {
	if runtime.GOOS != "windows" {
		return "scan: Windows Defender supported only on Windows."
	}

	scanType, mpScanType, ok := parseScanArgs(args)
	if !ok {
		return "scan: unrecognized target. Use `scan system` or `scan system full`."
	}

	out, err := runDefenderPowershellScanBlocking(scanType, 20*time.Minute)
	if err == nil {
		trim := strings.TrimSpace(out)
		if trim == "" {
			return fmt.Sprintf("Windows Defender %s completed. No output.", scanType)
		}
		return fmt.Sprintf("Windows Defender %s completed:\n%s", scanType, trim)
	}

	out2, err2 := runMpCmdRunScanBlocking(mpScanType, 20*time.Minute)
	if err2 == nil {
		trim := strings.TrimSpace(out2)
		if trim == "" {
			return fmt.Sprintf("Windows Defender (MpCmdRun) %s completed. No output.", scanType)
		}
		return fmt.Sprintf("Windows Defender (MpCmdRun) %s completed:\n%s", scanType, trim)
	}

	return fmt.Sprintf("scan failed.\nPowerShell err: %v\nMpCmdRun err: %v", err, err2)
}

func parseScanArgs(args []string) (psScanType string, mpScanType string, ok bool) {
	psScanType = "QuickScan"
	mpScanType = "1"
	ok = true

	if len(args) == 0 {
		return psScanType, mpScanType, ok
	}

	first := strings.ToLower(args[0])
	if first == "system" {
		if len(args) > 1 {
			switch strings.ToLower(args[1]) {
			case "full", "fullscan", "full-scan":
				return "FullScan", "2", true
			case "quick", "quickscan", "quick-scan":
				return "QuickScan", "1", true
			}
		}
		return "QuickScan", "1", true
	}

	switch first {
	case "full", "fullscan", "full-scan":
		return "FullScan", "2", true
	case "quick", "quickscan", "quick-scan":
		return "QuickScan", "1", true
	default:
		return "", "", false
	}
}

func runDefenderPowershellScanStream(scanType string, ch chan string, timeout time.Duration) error {
	psCmd := fmt.Sprintf(`
Try {
    Import-Module Defender -ErrorAction SilentlyContinue;
    Start-MpScan -ScanType %s -ErrorAction Stop;
    $t = Get-MpThreatDetection -ErrorAction SilentlyContinue;
    if ($t -and $t.Count -gt 0) {
        $t | Format-Table -AutoSize | Out-String;
    } else {
        Write-Output 'No threats found.';
    }
} Catch {
    Write-Error $_.Exception.Message;
    exit 2
}`, scanType)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	errCh := make(chan error, 2)
	go streamReaderToChan(stdout, ch, errCh)
	go streamReaderToChan(stderr, ch, errCh)

	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()

	select {
	case err := <-waitCh:
		select {
		case rerr := <-errCh:
			if rerr != nil {
				//non-fatal:continue to return process error if any
			}
		default:
		}
		if err != nil {
			return err
		}
		return nil
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		return errors.New("powershell scan timed out")
	}
}

func runMpCmdRunScanStream(scanType string, ch chan string, timeout time.Duration) error {
	args := []string{"-Scan", "-ScanType", scanType}
	tryPaths := []string{
		"MpCmdRun.exe",
		`C:\Program Files\Windows Defender\MpCmdRun.exe`,
		`C:\Program Files\Microsoft Defender ATP\MpCmdRun.exe`,
	}

	var lastErr error
	for _, exe := range tryPaths {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		cmd := exec.CommandContext(ctx, exe, args...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			cancel()
			lastErr = err
			continue
		}

		if err := cmd.Start(); err != nil {
			cancel()
			lastErr = err
			continue
		}

		errCh := make(chan error, 2)
		go streamReaderToChan(stdout, ch, errCh)
		go streamReaderToChan(stderr, ch, errCh)

		waitCh := make(chan error, 1)
		go func() { waitCh <- cmd.Wait() }()

		select {
		case err := <-waitCh:
			cancel()
			if err != nil {
				lastErr = fmt.Errorf("%s: %v", exe, err)
				continue
			}
			return nil
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			cancel()
			lastErr = fmt.Errorf("%s: timed out", exe)
			continue
		}
	}
	if lastErr == nil {
		lastErr = errors.New("MpCmdRun not found")
	}
	return lastErr
}

func streamReaderToChan(r io.Reader, ch chan string, errCh chan error) {
	sc := bufio.NewScanner(r)
	const maxToken = 1024 * 16
	buf := make([]byte, maxToken)
	sc.Buffer(buf, maxToken)
	for sc.Scan() {
		line := sc.Text()
		line = strings.TrimRight(line, "\r\n")
		if line != "" {
			ch <- line
		}
	}
	if err := sc.Err(); err != nil {
		errCh <- err
		return
	}
	errCh <- nil
}

func runDefenderPowershellScanBlocking(scanType string, timeout time.Duration) (string, error) {
	psCmd := fmt.Sprintf(`
Try {
    Import-Module Defender -ErrorAction SilentlyContinue;
    Start-MpScan -ScanType %s -ErrorAction Stop;
    $t = Get-MpThreatDetection -ErrorAction SilentlyContinue;
    if ($t -and $t.Count -gt 0) {
        $t | Format-Table -AutoSize | Out-String;
    } else {
        Write-Output 'No threats found.';
    }
} Catch {
    Write-Error $_.Exception.Message;
    exit 2
}`, scanType)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", psCmd)
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		c := strings.TrimSpace(stderr.String())
		if c == "" {
			c = err.Error()
		}
		return "", fmt.Errorf("powershell: %s", c)
	}
	return out.String(), nil
}

func runMpCmdRunScanBlocking(scanType string, timeout time.Duration) (string, error) {
	args := []string{"-Scan", "-ScanType", scanType}
	tryPaths := []string{
		"MpCmdRun.exe",
		`C:\Program Files\Windows Defender\MpCmdRun.exe`,
		`C:\Program Files\Microsoft Defender ATP\MpCmdRun.exe`,
	}

	var lastErr error
	for _, exe := range tryPaths {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		cmd := exec.CommandContext(ctx, exe, args...)
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			cancel()
			lastErr = fmt.Errorf("%s: %s", exe, strings.TrimSpace(stderr.String()))
			continue
		}
		cancel()
		combined := strings.TrimSpace(out.String() + "\n" + stderr.String())
		return combined, nil
	}
	if lastErr == nil {
		lastErr = errors.New("MpCmdRun not found")
	}
	return "", lastErr
}
