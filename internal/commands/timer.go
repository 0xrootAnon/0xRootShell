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
	"strconv"
	"strings"
	"time"
)

func ScheduleTimer(args []string, ch chan string) {
	if ch == nil {
		return
	}
	if len(args) == 0 {
		ch <- "timer: expected duration like '25m' or time like '0630'"
		return
	}
	raw := args[0]
	if d, err := time.ParseDuration(raw); err == nil {
		ch <- fmt.Sprintf("Timer set for %s from now.", d.String())
		time.AfterFunc(d, func() {
			ch <- fmt.Sprintf("Timer: %s elapsed.", d.String())
		})
		return
	}
	s := strings.ReplaceAll(raw, ":", "")
	if len(s) == 4 {
		hh, _ := strconv.Atoi(s[:2])
		mm, _ := strconv.Atoi(s[2:])
		now := time.Now()
		target := time.Date(now.Year(), now.Month(), now.Day(), hh, mm, 0, 0, now.Location())
		if target.Before(now) {
			target = target.Add(24 * time.Hour)
		}
		ch <- fmt.Sprintf("Alarm set for %s", target.Format("2006-01-02 15:04"))
		duration := target.Sub(now)
		time.AfterFunc(duration, func() {
			ch <- fmt.Sprintf("Alarm: %s reached.", target.Format("2006-01-02 15:04"))
		})
		return
	}
	ch <- "timer: unrecognized format. Try '25m' or '0630'."
}
