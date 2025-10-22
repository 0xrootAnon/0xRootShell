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

type Goal struct {
	ID      int    `json:"id"`
	Text    string `json:"text"`
	Created string `json:"created"`
	Done    bool   `json:"done"`
}

func goalsFilePath() string {
	_ = os.MkdirAll("data", 0755)
	return filepath.Join("data", "goals.json")
}

func loadGoals() ([]Goal, error) {
	fp := goalsFilePath()
	b, err := os.ReadFile(fp)
	if err != nil {
		if os.IsNotExist(err) {
			return []Goal{}, nil
		}
		return nil, err
	}
	var gs []Goal
	if err := json.Unmarshal(b, &gs); err != nil {
		return nil, err
	}
	return gs, nil
}

func saveGoals(gs []Goal) error {
	fp := goalsFilePath()
	b, err := json.MarshalIndent(gs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fp, b, 0644)
}

func nextGoalID(gs []Goal) int {
	max := 0
	for _, g := range gs {
		if g.ID > max {
			max = g.ID
		}
	}
	return max + 1
}

func CmdGoal(args []string) string {
	if len(args) == 0 {
		return `goal: subcommands: add, list, done, remove, clear
Examples:
  goal add "learn go"
  goal list
  goal done 1
  goal remove 2
  goal clear --confirm`
	}

	sub := strings.ToLower(args[0])

	switch sub {
	case "add":
		if len(args) < 2 {
			return "goal add: expected text, e.g. goal add \"learn go\""
		}
		text := strings.Join(args[1:], " ")
		text = strings.TrimSpace(text)
		if text == "" {
			return "goal add: empty text"
		}
		gs, err := loadGoals()
		if err != nil {
			return "goal add: load error: " + err.Error()
		}
		id := nextGoalID(gs)
		g := Goal{ID: id, Text: text, Created: time.Now().UTC().Format(time.RFC3339), Done: false}
		gs = append(gs, g)
		if err := saveGoals(gs); err != nil {
			return "goal add: save error: " + err.Error()
		}
		return fmt.Sprintf("Added goal #%d: %s", id, text)

	case "list":
		gs, err := loadGoals()
		if err != nil {
			return "goal list: load error: " + err.Error()
		}
		if len(gs) == 0 {
			return "No goals. Add one with: goal add \"learn go\""
		}
		sb := &strings.Builder{}
		for _, g := range gs {
			check := "[ ]"
			if g.Done {
				check = "[x]"
			}
			created := g.Created
			if t, err := time.Parse(time.RFC3339, g.Created); err == nil {
				created = t.Local().Format("2006-01-02 15:04")
			}
			fmt.Fprintf(sb, "%d) %s %s  — %s\n", g.ID, check, g.Text, created)
		}
		return strings.TrimSpace(sb.String())

	case "done":
		if len(args) < 2 {
			return "goal done: expected id, e.g. goal done 1"
		}
		id, err := strconv.Atoi(args[1])
		if err != nil {
			return "goal done: invalid id"
		}
		gs, err := loadGoals()
		if err != nil {
			return "goal done: load error: " + err.Error()
		}
		found := false
		for i := range gs {
			if gs[i].ID == id {
				gs[i].Done = true
				found = true
				break
			}
		}
		if !found {
			return fmt.Sprintf("goal done: id %d not found", id)
		}
		if err := saveGoals(gs); err != nil {
			return "goal done: save error: " + err.Error()
		}
		return fmt.Sprintf("Marked goal %d done.", id)

	case "remove", "rm", "delete":
		if len(args) < 2 {
			return "goal remove: expected id, e.g. goal remove 1"
		}
		id, err := strconv.Atoi(args[1])
		if err != nil {
			return "goal remove: invalid id"
		}
		gs, err := loadGoals()
		if err != nil {
			return "goal remove: load error: " + err.Error()
		}
		newGs := make([]Goal, 0, len(gs))
		found := false
		for _, g := range gs {
			if g.ID == id {
				found = true
				continue
			}
			newGs = append(newGs, g)
		}
		if !found {
			return fmt.Sprintf("goal remove: id %d not found", id)
		}
		if err := saveGoals(newGs); err != nil {
			return "goal remove: save error: " + err.Error()
		}
		return fmt.Sprintf("Removed goal %d.", id)

	case "clear":
		if len(args) >= 2 && (args[1] == "--confirm" || args[1] == "confirm") {
			fp := goalsFilePath()
			_ = os.Remove(fp)
			return "All goals cleared."
		}
		return "goal clear: destructive. confirm with: goal clear --confirm"

	default:
		return "goal: unknown subcommand. Try 'goal add', 'goal list', 'goal done <id>'"
	}
}
