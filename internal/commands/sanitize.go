// internal/commands/sanitize.go
package commands

import (
	"regexp"
	"strings"
)

// simple compiled regex for common ANSI CSI sequences like \x1b[31m etc.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

// sanitizeOutput removes CRs and ANSI escape sequences so the UI won't
// mis-handle carriage returns or color codes when printing char-by-char.
func sanitizeOutput(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = ansiRe.ReplaceAllString(s, "")
	return s
}
