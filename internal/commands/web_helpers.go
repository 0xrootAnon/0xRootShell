package commands

// Small wrappers that reuse your existing CmdSearch (already present in media.go).
// They build a query prefix (like "weather <args>") and call your project's CmdSearch,
// avoiding any duplicate CmdSearch symbol.

func CmdWeather(args []string) string {
	// call existing CmdSearch with "weather <args...>"
	q := append([]string{"weather"}, args...)
	// CmdSearch exists in your media.go, so just call it
	return CmdSearch(q)
}

func CmdConvert(args []string) string {
	// "convert" queries are usually like "100 usd to inr"
	// pass through to CmdSearch
	if len(args) == 0 {
		return "convert: expected query, e.g. `convert 100 usd to inr`"
	}
	return CmdSearch(args)
}

func CmdNews(args []string) string {
	if len(args) == 0 {
		// general news
		return CmdSearch([]string{"news"})
	}
	q := append([]string{"news"}, args...)
	return CmdSearch(q)
}
