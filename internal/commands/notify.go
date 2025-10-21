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
	"strings"
	"time"
)

type OutgoingMsg struct {
	ID        string    `json:"id"`
	To        string    `json:"to"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"ts"`
}

func messagesFilePath() string {
	_ = os.MkdirAll("data/outgoing_messages", 0755)
	return filepath.Join("data", "outgoing_messages", "messages.json")
}

func appendMessage(m OutgoingMsg) error {
	fn := messagesFilePath()
	var arr []OutgoingMsg
	if b, err := os.ReadFile(fn); err == nil {
		_ = json.Unmarshal(b, &arr)
	}
	arr = append(arr, m)
	nb, _ := json.MarshalIndent(arr, "", "  ")
	return os.WriteFile(fn, nb, 0644)
}

func CmdMessage(args []string) string {
	if len(args) < 2 {
		return "message: expected 'message send <contact> \"text\"'"
	}
	if strings.ToLower(args[0]) == "send" {
		to := args[1]
		text := strings.Join(args[2:], " ")
		text = strings.Trim(text, "\"")
		msg := OutgoingMsg{
			ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
			To:        to,
			Text:      text,
			Timestamp: time.Now().UTC(),
		}
		if err := appendMessage(msg); err != nil {
			return "message: save error: " + err.Error()
		}
		return fmt.Sprintf("Message queued to %s: %s", to, text)
	}
	return "message: unknown subcommand"
}

func CmdNotify(args []string) string {
	if len(args) == 0 {
		return "notify: expected subcommand list/send"
	}
	switch strings.ToLower(args[0]) {
	case "list":
		fn := messagesFilePath()
		if b, err := os.ReadFile(fn); err == nil {
			return string(b)
		}
		return "notify: none"
	case "send":
		text := strings.Join(args[1:], " ")
		text = strings.Trim(text, "\"")
		msg := OutgoingMsg{
			ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
			To:        "local",
			Text:      text,
			Timestamp: time.Now().UTC(),
		}
		if err := appendMessage(msg); err != nil {
			return "notify send error: " + err.Error()
		}
		return "Notification saved."
	default:
		return "notify: unknown subcommand"
	}
}

func CmdMail(args []string) string {
	if len(args) == 0 {
		return "mail: expected 'check' or 'open' or 'compose'"
	}
	switch strings.ToLower(args[0]) {
	case "check":
		return "mail: mail integration not configured. Consider connecting an email plugin in plugins/."
	case "open":
		if err := runOpen("mailto:"); err != nil {
			return "mail open error: " + err.Error()
		}
		return "Opened mail client"
	default:
		return "mail: unknown subcommand"
	}
}
