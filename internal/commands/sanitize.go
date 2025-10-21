package commands

import (
	"regexp"
	"strings"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func sanitizeOutput(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = ansiRe.ReplaceAllString(s, "")
	return s
}
