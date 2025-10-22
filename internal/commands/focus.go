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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type focusState struct {
	EndUnix int64  `json:"end_unix"`
	Label   string `json:"label,omitempty"`
}

func focusStatePath() string {
	_ = os.MkdirAll("data", 0755)
	return filepath.Join("data", "focus_state.json")
}

func readFocusState() (focusState, bool, error) {
	fp := focusStatePath()
	b, err := os.ReadFile(fp)
	if err != nil {
		if os.IsNotExist(err) {
			return focusState{}, false, nil
		}
		return focusState{}, false, err
	}
	var s focusState
	if err := json.Unmarshal(b, &s); err != nil {
		return focusState{}, false, err
	}
	if s.EndUnix <= 0 {
		return focusState{}, false, nil
	}
	return s, true, nil
}

func writeFocusState(s focusState) error {
	fp := focusStatePath()
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fp, b, 0644)
}

func removeFocusStateFile() error {
	fp := focusStatePath()
	if err := os.Remove(fp); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}

func parseDurationLike(tokens []string) (time.Duration, string, error) {
	if len(tokens) == 0 {
		return 0, "", fmt.Errorf("no duration provided")
	}
	joined := strings.Join(tokens, " ")
	joined = strings.TrimSpace(joined)

	if strings.Contains(joined, "min") || strings.Contains(joined, "m") || strings.Contains(joined, "h") {
		norm := strings.ReplaceAll(joined, "min", "m")
		if d, err := time.ParseDuration(norm); err == nil {
			return d, norm, nil
		}
	}

	if n, err := strconv.Atoi(strings.Fields(joined)[0]); err == nil {
		return time.Duration(n) * time.Minute, fmt.Sprintf("%dm", n), nil
	}

	if d, err := time.ParseDuration(joined); err == nil {
		return d, joined, nil
	}

	return 0, "", fmt.Errorf("could not parse duration: %s", joined)
}

func StartFocus(args []string, ch chan string) {
	if len(args) == 0 {
		ch <- "focus: expected duration (e.g. focus 25m or focus 90 min) or 'focus end'."
		return
	}
	if strings.ToLower(args[0]) == "end" {
		if err := removeFocusStateFile(); err != nil {
			ch <- "focus: failed to end: " + err.Error()
			return
		}
		ch <- "Focus ended."
		return
	}

	dur, norm, err := parseDurationLike(args)
	if err != nil {
		ch <- "focus: " + err.Error()
		return
	}
	if dur <= 0 {
		ch <- "focus: duration must be > 0"
		return
	}

	if st, ok, _ := readFocusState(); ok {
		et := time.Unix(st.EndUnix, 0)
		if time.Now().Before(et) {
			ch <- fmt.Sprintf("A focus session is already running and ends at %s (use 'focus end' to stop it).", et.Local().Format("2006-01-02 15:04"))
			return
		}
	}

	end := time.Now().Add(dur)
	state := focusState{EndUnix: end.Unix()}
	if err := writeFocusState(state); err != nil {
		ch <- "focus: failed to create state: " + err.Error()
		return
	}

	ch <- fmt.Sprintf("Focus started for %s — ends at %s", norm, end.Local().Format("15:04"))

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if st, ok, _ := readFocusState(); !ok {
				ch <- "Focus ended (cancelled)."
				return
			} else {
				if time.Now().Unix() >= st.EndUnix {
					_ = removeFocusStateFile() // cleanup
					ch <- "Focus session complete! Great job."
					return
				}
			}
		}
	}
}

func EndFocus() string {
	if err := removeFocusStateFile(); err != nil {
		return "focus: failed to end: " + err.Error()
	}
	return "Focus ended."
}

func CmdFocus(args []string) string {
	if len(args) == 0 {
		return "focus: usage: focus <duration>  (e.g. focus 25m, focus 90 min) or focus end"
	}
	if strings.ToLower(args[0]) == "end" {
		return EndFocus()
	}

	dur, norm, err := parseDurationLike(args)
	if err != nil {
		return "focus: " + err.Error()
	}
	if dur <= 0 {
		return "focus: duration must be > 0"
	}

	end := time.Now().Add(dur)
	state := focusState{EndUnix: end.Unix()}
	if err := writeFocusState(state); err != nil {
		return "focus: failed to create state: " + err.Error()
	}
	return fmt.Sprintf("Focus started for %s — ends at %s", norm, end.Local().Format("15:04"))
}
