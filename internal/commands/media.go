package commands

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/skratchdot/open-golang/open"
)

func CmdPlay(args []string) string {
	if len(args) == 0 {
		return "play: expected 'music <name>' or 'youtube <query>' or a file/url"
	}
	sub := strings.ToLower(args[0])
	rest := strings.Join(args[1:], " ")
	switch sub {
	case "youtube", "yt":
		if rest == "" {
			return "play youtube: expected search query"
		}
		q := url.QueryEscape(rest)
		open.Run("https://www.youtube.com/results?search_query=" + q)
		return fmt.Sprintf("Searching YouTube for: %s", rest)
	case "music":
		if rest == "" {
			return "play music: expected query"
		}
		q := url.QueryEscape(rest)
		open.Run("https://open.spotify.com/search/" + q)
		return fmt.Sprintf("Opening Spotify search: %s", rest)
	default:
		target := strings.Join(args, " ")
		if err := open.Run(target); err == nil {
			return "Playing/opening: " + target
		}
		return "play: couldn't open target. If it's a local file, provide full path."
	}
}

func CmdSearch(args []string) string {
	if len(args) == 0 {
		return "search: expected query"
	}
	q := url.QueryEscape(strings.Join(args, " "))
	open.Run("https://www.google.com/search?q=" + q)
	return "Opened browser search."
}
