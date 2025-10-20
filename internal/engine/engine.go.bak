package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xrootAnon/0xRootShell/internal/commands"
	"github.com/0xrootAnon/0xRootShell/internal/store"
)

type Engine struct {
	store *store.Store
	cwd   string //current working directory for the session
}

func NewEngine(s *store.Store) *Engine {
	wd, err := os.Getwd()
	if err != nil {
		wd = "." //fallback
	}
	return &Engine{store: s, cwd: wd}
}

// parses the input and dispatches to the appropriate handler.
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
		return commands.CmdFind(args)
	case "sys":
		return commands.CmdSys(args) // new: sys subcommands
	case "audio", "vol":
		return commands.CmdAudio(args) // new
	case "display", "brightness", "screen":
		return commands.CmdDisplay(args) // new
	case "net", "network":
		return commands.CmdNet(args) // new
	case "file", "files":
		return commands.CmdFile(args) // new
	case "compress", "zip", "extract":
		return commands.CmdCompressArchive(append([]string{verb}, args...)) // compress/extract handler
	case "screenshot":
		return commands.CmdScreenshot(args)
	case "play":
		return commands.CmdPlay(args)
	case "search", "web":
		return commands.CmdSearch(args)
	case "remind":
		return commands.CmdRemind(args) // simple local reminder persistence
	case "help":
		return helpText()
	case "history":
		h, _ := e.store.ListHistory(30)
		return strings.Join(h, "\n")
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
  find <pattern>           Fuzzy file search
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

// naive splitting by spaces but preserves quoted sequences
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
	return "" //success -> empty output (shell convention); can return path or a confirmation
}
