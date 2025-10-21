package commands

func CmdWeather(args []string) string {
	q := append([]string{"weather"}, args...)
	return CmdSearch(q)
}

func CmdConvert(args []string) string {
	if len(args) == 0 {
		return "convert: expected query, e.g. `convert 100 usd to inr`"
	}
	return CmdSearch(args)
}

func CmdNews(args []string) string {
	if len(args) == 0 {
		return CmdSearch([]string{"news"})
	}
	q := append([]string{"news"}, args...)
	return CmdSearch(q)
}
