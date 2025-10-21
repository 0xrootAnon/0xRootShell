package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/0xrootAnon/0xRootShell/internal/commands"
	"github.com/0xrootAnon/0xRootShell/internal/store"
)

type Engine struct {
	store   *store.Store
	cwd     string // current working directory for the session
	MsgChan chan string
}

// sanitizeForUI removes CRs and ANSI sequences before sending to the UI.
func sanitizeForUI(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	return ansi.ReplaceAllString(s, "")
}

// NewEngine now accepts a message channel the engine can use to send asynchronous messages
// back to the UI (e.g., background find results).
func NewEngine(s *store.Store, ch chan string) *Engine {
	wd, err := os.Getwd()
	if err != nil {
		wd = "." // fallback
	}
	return &Engine{store: s, cwd: wd, MsgChan: ch}
}

// Execute parses the input and dispatches to the appropriate handler.
// For potentially long-running commands like "find" we run them in a goroutine
// and send results back over the engine message channel so the UI remains responsive.
func (e *Engine) Execute(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parts := splitArgs(raw)
	verb := strings.ToLower(parts[0])
	args := parts[1:]

	switch verb {
	case "cd":
		return e.CmdCd(args)
	case "pwd":
		return e.CmdPwd()
	case "launch", "openapp", "start":
		return commands.CmdLaunch(args)
	case "open":
		return commands.CmdOpen(args)
	case "find", "searchfile":
		// run find in background and send results back to UI via MsgChan
		if e.MsgChan != nil {
			// capture values for goroutine
			qargs := append([]string(nil), args...)
			go func(a []string) {
				// initial notice
				query := strings.Join(a, " ")
				if query == "" {
					query = "(empty)"
				}
				e.MsgChan <- sanitizeForUI(fmt.Sprintf("Searching for: %s", query))
				// run the existing synchronous CmdFind (safe reuse)
				res := commands.CmdFind(a)
				// ensure results are clearly demarcated
				e.MsgChan <- sanitizeForUI(fmt.Sprintf("=== Search results for: %s ===\n%s\n=== End results ===", query, res))
			}(qargs)
			return "Searching... results will appear below when ready."
		}
		// fallback synchronous behavior if no message channel
		return commands.CmdFind(args)
	case "sys":
		return commands.CmdSys(args)
	case "audio", "vol":
		return commands.CmdAudio(args)
	case "display", "brightness", "screen":
		return commands.CmdDisplay(args)
	case "net", "network":
		return commands.CmdNet(args)
	case "file", "files":
		return commands.CmdFile(args)
	case "compress", "zip", "extract":
		return commands.CmdCompressArchive(append([]string{verb}, args...))
	case "screenshot":
		return commands.CmdScreenshot(args)
	case "search", "web":
		return commands.CmdSearch(args)
	case "remind":
		return commands.CmdRemind(args)
	case "help":
		return helpText()
	case "history":
		h, _ := e.store.ListHistory(30)
		return strings.Join(h, "\n")
	case "weather":
		return commands.CmdWeather(args) // opens browser with weather search
	case "convert", "currency":
		return commands.CmdConvert(args) // currency conversion via web search
	case "news":
		return commands.CmdNews(args) // open news search
	case "message", "msg":
		return commands.CmdMessage(args) // local/outgoing-message queue or plugin hook
	case "mail":
		return commands.CmdMail(args) // placeholder / open mail client
	case "notify":
		return commands.CmdNotify(args) // list / create notifications (local)
	case "play":
		return commands.CmdPlay(args) // play file/url; spawns ffmpeg/player if needed
	case "pause", "next", "prev":
		return commands.CmdMediaControl(append([]string{verb}, args...))
	case "record":
		// run screen/cam recording in background and report when done
		if e.MsgChan != nil {
			qargs := append([]string(nil), args...)
			go func(a []string) {
				e.MsgChan <- "Starting recording..."
				res := commands.CmdRecord(a) // new commands file; returns result message
				e.MsgChan <- res
			}(qargs)
			return "Recording started..."
		}
		return commands.CmdRecord(args)
	case "alarm", "timer":
		// schedule a timer/alarm in background using engine message channel
		if e.MsgChan != nil {
			go commands.ScheduleTimer(args, e.MsgChan)
			return "Timer scheduled."
		}
		return "Timer not scheduled: no message channel."
	case "speedtest":
		return commands.CmdSpeedtest(args)
	case "ls":
		return commands.CmdLS(args)
	case "calc":
		return commands.CmdCalc(args)
	default:
		return "Unknown command. Try 'help'."
	}
}

func helpText() string {
	return `0xRootShell — help
Common commands:
  cd <dir>                 Change current directory
  pwd                      Print current directory
  launch <app>             Launch application or URL
  open <file|url>          Open file or URL
  find <pattern>           Fuzzy file search (non-blocking; results appear when ready)
  sys <status|lock|sleep|off|bootlog|update>
  audio vol <0-100>        Set volume, audio mute/unmute
  display bright <0-100>   Set screen brightness
  net wifi <on|off|list>   Manage wifi (nmcli/netsh)
  file move <src> <dst>
  compress <zip> <src>     Create zip archive
  extract <zip> <dst>      Extract zip
  screenshot               Save a screenshot to data/screenshots/
  search <query>           Open browser with Google search
  play <youtube|file> ...  Play media / open URL
  remind <text>            Save a quick reminder (stored locally)
  history
  help
`
}

// The rest of the engine code (splitArgs, CmdCd, CmdPwd, etc.) remains unchanged.
// We'll include the existing implementations for splitArgs, CmdPwd, and CmdCd below,
// copied from your previous engine implementation for completeness.

func splitArgs(s string) []string {
	var out []string
	var cur strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '"' {
			inQuote = !inQuote
			continue
		}
		if c == ' ' && !inQuote {
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteByte(c)
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

/***dir commands ***/

// returns the engine's current working directory.
func (e *Engine) CmdPwd() string {
	//use filepath.Clean for consistent separators
	return filepath.Clean(e.cwd)
}

// Supports:
//
//	cd relative/path
//	cd /absolute/path
//	cd ~ (home dir)
//	cd .. etc.
func (e *Engine) CmdCd(args []string) string {
	target := ""
	if len(args) == 0 || args[0] == "" {
		//no args -> go to home dir
		home, err := os.UserHomeDir()
		if err != nil {
			return "cd: cannot determine home directory"
		}
		target = home
	} else {
		target = args[0]
	}

	//expand ~
	if strings.HasPrefix(target, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "cd: cannot determine home directory"
		}
		if target == "~" {
			target = home
		} else if strings.HasPrefix(target, "~/") || strings.HasPrefix(target, `~\`) {
			target = filepath.Join(home, target[2:]) // remove "~/"
		}
	}

	//if relative path -> join with current cwd
	if !filepath.IsAbs(target) {
		target = filepath.Join(e.cwd, target)
	}

	//clean path
	target = filepath.Clean(target)

	//check existence and directory
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Sprintf("cd: %s: %v", target, err)
	}
	if !info.IsDir() {
		return fmt.Sprintf("cd: %s: not a directory", target)
	}

	//attempt to change process working directory
	if err := os.Chdir(target); err != nil {
		return fmt.Sprintf("cd: failed to change directory: %v", err)
	}

	//update engine cwd
	e.cwd = target
	return "" //success -> empty output (shell convention)
}
