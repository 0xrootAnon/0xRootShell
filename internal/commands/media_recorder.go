package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// CmdMediaControl: best-effort stub for pause/next/prev (not a universal API).
// This avoids duplicating your existing media controls (if any).
func CmdMediaControl(args []string) string {
	if len(args) == 0 {
		return "media control: expected pause/next/prev"
	}
	verb := strings.ToLower(args[0])
	switch verb {
	case "pause", "play", "toggle":
		return "media control: use your player-specific controls (no universal control implemented)."
	case "next", "prev":
		return "media control: next/prev not implemented universally. Use your media player controls."
	default:
		return "media control: unknown subcommand"
	}
}

// CmdRecord: record screen or webcam using ffmpeg if available.
// Usage: record screen 10s  OR record cam 15s  OR record screen out.mp4
// Note: This does only the recording logic; it does not override any existing CmdPlay/CmdSearch.
func CmdRecord(args []string) string {
	if len(args) == 0 {
		return "record: expected 'screen' or 'cam' and optional duration (seconds) or output filename"
	}
	mode := strings.ToLower(args[0])
	duration := 0
	outName := ""

	// heuristic parse of rest of args
	for _, a := range args[1:] {
		if strings.HasSuffix(a, ".mp4") || strings.HasSuffix(a, ".mkv") || strings.HasSuffix(a, ".webm") {
			outName = a
			continue
		}
		// try parse integer seconds
		var secs int
		if n, err := fmt.Sscanf(a, "%d", &secs); n == 1 && err == nil {
			duration = secs
		}
	}
	if outName == "" {
		now := time.Now().Format("20060102-150405")
		if mode == "cam" {
			outName = filepath.Join("data", "recordings", "cam-"+now+".mp4")
		} else {
			outName = filepath.Join("data", "recordings", "screen-"+now+".mp4")
		}
	}

	_ = os.MkdirAll(filepath.Dir(outName), 0755)

	if p, err := exec.LookPath("ffmpeg"); err == nil {
		var cmd *exec.Cmd
		if mode == "cam" {
			if runtime.GOOS == "windows" {
				// device name may vary; this is best-effort placeholder
				cmd = exec.Command(p, "-f", "dshow", "-i", "video=Integrated Camera", "-t", fmt.Sprintf("%d", duration), outName)
			} else {
				cmd = exec.Command(p, "-f", "v4l2", "-i", "/dev/video0", "-t", fmt.Sprintf("%d", duration), outName)
			}
		} else {
			// screen recording
			if runtime.GOOS == "windows" {
				args := []string{"-f", "gdigrab", "-framerate", "20", "-i", "desktop"}
				if duration > 0 {
					args = append(args, "-t", fmt.Sprintf("%d", duration))
				}
				args = append(args, outName)
				cmd = exec.Command(p, args...)
			} else if runtime.GOOS == "darwin" {
				args := []string{"-f", "avfoundation", "-i", "1:none"}
				if duration > 0 {
					args = append(args, "-t", fmt.Sprintf("%d", duration))
				}
				args = append(args, outName)
				cmd = exec.Command(p, args...)
			} else {
				display := os.Getenv("DISPLAY")
				if display == "" {
					display = ":0.0"
				}
				args := []string{"-f", "x11grab", "-framerate", "20", "-i", display}
				if duration > 0 {
					args = append(args, "-t", fmt.Sprintf("%d", duration))
				}
				args = append(args, outName)
				cmd = exec.Command(p, args...)
			}
		}
		if err := cmd.Start(); err != nil {
			return "record start error: " + err.Error()
		}
		return fmt.Sprintf("Recording started -> %s (ffmpeg PID %d)", outName, cmd.Process.Pid)
	}

	return "record: ffmpeg not found. Install ffmpeg to enable recording (https://ffmpeg.org/download.html)."
}
