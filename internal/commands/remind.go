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

type Reminder struct {
	ID      string    `json:"id"`
	Text    string    `json:"text"`
	Due     time.Time `json:"due,omitempty"`
	Created time.Time `json:"created"`
}

const remindersFile = "data/reminders.json"

// loadReminders reads reminders file (returns empty list if missing)
func loadReminders() ([]Reminder, error) {
	path := remindersFile
	// ensure data dir exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Reminder{}, nil
		}
		return nil, err
	}
	var r []Reminder
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	return r, nil
}

func saveReminders(r []Reminder) error {
	path := remindersFile
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

// parseOptionalTime attempts to parse the last token as a datetime in "2006-01-02" or "2006-01-02 15:04" formats.
// Returns (time, consumed) where consumed==true if a time token was parsed.
func parseOptionalTime(tokens []string) (time.Time, bool) {
	if len(tokens) == 0 {
		return time.Time{}, false
	}
	last := tokens[len(tokens)-1]
	// try "YYYY-MM-DD" first
	if t, err := time.Parse("2006-01-02", last); err == nil {
		return t, true
	}
	// try "YYYY-MM-DD_HH:MM" or "YYYY-MM-DD HH:MM"
	if strings.Contains(last, "_") {
		last = strings.ReplaceAll(last, "_", " ")
	}
	if t, err := time.Parse("2006-01-02 15:04", last); err == nil {
		return t, true
	}
	// try RFC3339
	if t, err := time.Parse(time.RFC3339, last); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// CmdRemind implements reminders CLI: add/list/rm/clear
func CmdRemind(args []string) string {
	if len(args) == 0 {
		// list by default (legacy friendly)
		return remindList()
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "add":
		if len(args) < 2 {
			return "remind add: expected text. Example: `remind add \"call mom\" 2025-10-21 20:00`"
		}
		// join rest tokens and optionally parse time
		toks := args[1:]
		due, consumed := parseOptionalTime(toks)
		text := strings.Join(toks, " ")
		if consumed {
			// remove last token from text
			text = strings.Join(toks[:len(toks)-1], " ")
		}
		return remindAdd(text, due)
	case "list":
		return remindList()
	case "rm", "del", "remove":
		if len(args) < 2 {
			return "remind rm: expected reminder ID. Use `remind list` to see IDs."
		}
		return remindRemove(args[1])
	case "clear":
		return remindClear()
	default:
		// fallback: treat as quick add: remind pay rent 2025-10-21 20:00
		// allow "remind pay rent" to add with no due
		toks := args
		due, consumed := parseOptionalTime(toks)
		text := strings.Join(toks, " ")
		if consumed {
			text = strings.Join(toks[:len(toks)-1], " ")
		}
		return remindAdd(text, due)
	}
}

func remindAdd(text string, due time.Time) string {
	if strings.TrimSpace(text) == "" {
		return "remind add: cannot add empty reminder"
	}
	rem, err := loadReminders()
	if err != nil {
		return "remind add: load error: " + err.Error()
	}
	id := strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	r := Reminder{
		ID:      id,
		Text:    text,
		Created: time.Now(),
	}
	if !due.IsZero() {
		r.Due = due
	}
	rem = append(rem, r)
	if err := saveReminders(rem); err != nil {
		return "remind add: save error: " + err.Error()
	}
	if !due.IsZero() {
		return fmt.Sprintf("Reminder saved: %s (due %s)", text, r.Due.Format("2006-01-02 15:04"))
	}
	return fmt.Sprintf("Reminder saved: %s", text)
}

func remindList() string {
	rem, err := loadReminders()
	if err != nil {
		return "remind list: load error: " + err.Error()
	}
	if len(rem) == 0 {
		return "No reminders."
	}
	// sort by due then created (simple)
	// (lightweight bubble-ish sort because list is small)
	for i := 0; i < len(rem)-1; i++ {
		for j := i + 1; j < len(rem); j++ {
			ti := rem[i].Due
			tj := rem[j].Due
			if ti.IsZero() && !tj.IsZero() {
				// j earlier
				rem[i], rem[j] = rem[j], rem[i]
			} else if !ti.IsZero() && !tj.IsZero() && rem[j].Due.Before(rem[i].Due) {
				rem[i], rem[j] = rem[j], rem[i]
			}
		}
	}
	sb := &strings.Builder{}
	for _, r := range rem {
		if r.Due.IsZero() {
			sb.WriteString(fmt.Sprintf("%s    %s\n", r.ID, r.Text))
		} else {
			sb.WriteString(fmt.Sprintf("%s    [%s] %s\n", r.ID, r.Due.Format("2006-01-02 15:04"), r.Text))
		}
	}
	return strings.TrimSpace(sb.String())
}

func remindRemove(id string) string {
	rem, err := loadReminders()
	if err != nil {
		return "remind rm: load error: " + err.Error()
	}
	newRem := []Reminder{}
	found := false
	for _, r := range rem {
		if r.ID == id {
			found = true
			continue
		}
		newRem = append(newRem, r)
	}
	if !found {
		return "remind rm: id not found"
	}
	if err := saveReminders(newRem); err != nil {
		return "remind rm: save error: " + err.Error()
	}
	return "Reminder removed: " + id
}

func remindClear() string {
	if err := saveReminders([]Reminder{}); err != nil {
		return "remind clear: error: " + err.Error()
	}
	return "All reminders cleared."
}
