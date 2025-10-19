package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// simple persistent reminders file in data/reminders.txt
func CmdRemind(args []string) string {
	if len(args) == 0 {
		// list reminders
		return remindList()
	}
	text := strings.TrimSpace(strings.Join(args, " "))
	if text == "" {
		return "remind: expected text"
	}
	dir := "data"
	_ = os.MkdirAll(dir, 0755)
	f := filepath.Join(dir, "reminders.txt")
	entry := fmt.Sprintf("%s\t%s\n", time.Now().Format(time.RFC3339), text)
	if err := os.WriteFile(f, []byte(entry), 0644); err != nil {
		// try append
		fh, err2 := os.OpenFile(f, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err2 != nil {
			return "remind: write error: " + err.Error()
		}
		defer fh.Close()
		fh.WriteString(entry)
	}
	return "Reminder saved."
}

func remindList() string {
	f := filepath.Join("data", "reminders.txt")
	if b, err := os.ReadFile(f); err == nil {
		return "Reminders:\n" + string(b)
	}
	return "No reminders."
}
