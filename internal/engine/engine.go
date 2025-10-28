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
	cwd     string
	MsgChan chan string
}

func sanitizeForUI(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	ansi := regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
	return ansi.ReplaceAllString(s, "")
}

func NewEngine(s *store.Store, ch chan string) *Engine {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	return &Engine{store: s, cwd: wd, MsgChan: ch}
}

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
	case "clear":
		if len(args) >= 2 && strings.ToLower(args[0]) == "browser" && strings.ToLower(args[1]) == "history" {
			return commands.CmdClearBrowserHistory(args[2:])
		}
		return "clear: unknown target. Try 'clear browser history' or use your terminal to clear the screen."
	case "browse":
		if len(args) > 0 && strings.ToLower(args[0]) == "private" {
			return commands.CmdBrowsePrivate(args[1:])
		}
		return "browse: unknown target. Try 'browse private'."
	case "open":
		return commands.CmdOpen(args)
	case "find", "searchfile":
		if e.MsgChan != nil {
			qargs := append([]string(nil), args...)
			go func(a []string) {
				query := strings.Join(a, " ")
				if query == "" {
					query = "(empty)"
				}
				e.MsgChan <- sanitizeForUI(fmt.Sprintf("Searching for: %s", query))
				res := commands.CmdFind(a)
				e.MsgChan <- sanitizeForUI(fmt.Sprintf("=== Search results for: %s ===\n%s\n=== End results ===", query, res))
			}(qargs)
			return "Searching... results will appear below when ready."
		}
		return commands.CmdFind(args)
	case "sys":
		if len(args) == 0 {
			return "sys: expected subcommand. Try 'sys status' or 'sys perf'"
		}
		sub := strings.ToLower(args[0])
		switch sub {
		case "status":
			return commands.CmdSysStatus()
		case "perf":
			return commands.CmdSysPerf()
		case "lock":
			return commands.SysLock()
		case "sleep":
			return commands.SysSleep()
		case "off", "shutdown":
			for _, a := range args {
				if a == "--confirm" || a == "-y" {
					return commands.SysShutdown()
				}
			}
			return "sys off: destructive action. append --confirm to actually shutdown."
		case "bootlog":
			return commands.SysBootLog()
		case "update":
			return "sys update: use OS update tool (Windows Settings / apt / dnf / etc.)"
		default:
			return "sys: unknown subcommand"
		}
	case "audio", "vol":
		return commands.CmdAudio(args)
	case "display", "brightness", "screen":
		return commands.CmdDisplay(args)
	case "net", "network":
		return commands.CmdNet(args)
	case "scan":
		if e.MsgChan != nil {
			go commands.StartScan(args, e.MsgChan)
			return "Scan started... results will appear below."
		}
		return commands.CmdScan(args)
	case "create":
		if len(args) > 0 {
			t := strings.ToLower(args[0])
			if t == "folder" || t == "directory" || t == "dir" {
				return commands.CmdMkdir(args[1:])
			}
			return commands.CmdMkdir(args)
		}
		return commands.CmdMkdir(args)
	case "touch":
		return commands.CmdTouch(args)
	case "new":
		if len(args) == 0 {
			return commands.CmdMkdir(args)
		}
		first := strings.ToLower(args[0])
		if first == "file" || first == "document" {
			return commands.CmdTouch(args[1:])
		}
		if first == "folder" || first == "directory" || first == "dir" {
			return commands.CmdMkdir(args[1:])
		}

		if len(args) > 0 {
			if ext := filepath.Ext(args[0]); ext != "" {
				return commands.CmdTouch(args)
			}
		}

		return commands.CmdMkdir(args)
	case "remove", "delete":
		if len(args) > 0 {
			t := strings.ToLower(args[0])
			if t == "folder" || t == "directory" || t == "dir" {
				return commands.CmdRmdir(args[1:])
			}
			if t == "file" {
				return commands.CmdDel(args[1:])
			}
			return commands.CmdDel(args)
		}
		return "remove: usage examples: 'remove folder <name>' or 'remove <file>'"

	case "mkdir":
		return commands.CmdMkdir(args)
	case "rmdir":
		return commands.CmdRmdir(args)
	case "del", "deletefile":
		return commands.CmdDel(args)
	case "rm":
		return commands.CmdRm(args)
	case "copy", "cp":
		return commands.CmdCp(args)
	case "move", "mv":
		return commands.CmdMv(args)
	case "view", "read", "openfile":
		return commands.CmdCat(args)
	case "cat":
		return commands.CmdCat(args)
	case "search-in", "searchinside", "findin":
		return commands.CmdGrep(args)
	case "grep":
		return commands.CmdGrep(args)
	case "tasks", "processes":
		return commands.CmdTasklist(args)
	case "tasklist":
		return commands.CmdTasklist(args)
	case "kill", "end", "terminate", "stop":
		return commands.CmdTaskkill(args)
	case "taskkill":
		return commands.CmdTaskkill(args)
	case "drives", "volumes", "disk", "disks":
		return commands.CmdGetVolume(args)
	case "get-volume", "wmic":
		return commands.CmdGetVolume(args)
	case "save":
		if len(args) > 0 && (strings.ToLower(args[0]) == "file" || strings.HasPrefix(args[0], ".") || filepath.Ext(args[0]) != "") {
			return commands.CmdTouch(args[1:])
		}
		return "save: try 'save file <name>'"
	case "file", "files":
		return commands.CmdFile(args)
	case "compress", "zip", "extract":
		return commands.CmdCompressArchive(append([]string{verb}, args...))
	case "screenshot":
		return commands.CmdScreenshot(args)
	case "search", "web":
		return commands.CmdSearch(args)
	case "show":
		if len(args) > 0 && strings.ToLower(args[0]) == "notifications" {
			return commands.CmdShowNotifications()
		}
		return "show: unknown target. Try 'show notifications'"
	case "remind":
		return commands.CmdRemind(args)
	case "goal":
		return commands.CmdGoal(args)
	case "focus":
		if e.MsgChan != nil {
			qargs := append([]string(nil), args...)
			go commands.StartFocus(qargs, e.MsgChan)
			return "Focus started..."
		}
		return commands.CmdFocus(args)
	case "help":
		return helpText()
	case "history":
		h, _ := e.store.ListHistory(30)
		return strings.Join(h, "\n")
	case "weather":
		return commands.CmdWeather(args)
	case "convert", "currency":
		return commands.CmdConvert(args)
	case "news":
		return commands.CmdNews(args)
	case "message", "msg":
		return commands.CmdMessage(args)
	case "mail":
		return commands.CmdMail(args)
	case "notify":
		return commands.CmdNotify(args)
	case "play":
		return commands.CmdPlay(args)
	case "pause", "next", "prev":
		return commands.CmdMediaControl(append([]string{verb}, args...))
	case "alarm", "timer":
		if e.MsgChan != nil {
			go commands.ScheduleTimer(args, e.MsgChan)
			return "Timer scheduled."
		}
		return "Timer not scheduled: no message channel."
	case "speedtest":
		if e.MsgChan != nil {
			qargs := append([]string(nil), args...)
			go func(a []string) {
				e.MsgChan <- sanitizeForUI("Starting speedtest...")
				commands.CmdSpeedtestStream(a, e.MsgChan)
				e.MsgChan <- sanitizeForUI("Speedtest finished.")
			}(qargs)
			return "Running speedtest... results will appear below."
		}
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
	return `				0xRootShell
Common commands:
  sys <lock|sleep|off|bootlog|update|status|perf>
  cd <dir>                		Change current directory
  pwd                      		Print current directory
  launch <app>             		Launch application or URL (aliases: openapp, start)
  open <file|url>          		Open file or URL
  clear browser history    		Clear browser history (use: 'clear browser history')
  browse private <query>   		Private browsing helper (use: 'browse private')
  find <pattern>           		Fuzzy file search (non-blocking; results appear when ready)
  scan  	             		Start system scan (non-blocking; results appear when ready) 
  audio vol <0-100>        		Set volume, audio mute/unmute
  display bright <0-100>   		Set screen brightness (aliases: display, brightness, screen)
  net wifi <on|off|list>   		Manage wifi (nmcli / netsh)
  file <subcmd>            		File operations (aliases: files) — e.g. file move <src> <dst>
  compress|zip <zip> <src>		Create zip archive
  extract <zip> <dst>      		Extract zip
  screenshot               		Save a screenshot to data/screenshots/
  sys perf                  	Full system performance (CPU/Memory/Disk/Top processes)
  show notifications       		Show saved notifications
  search|web <query>       		Open browser with Google search
  play <youtube|file|url>  		Play media / open URL
  pause|next|prev          		Media controls (pause / next / prev)
  remind <text>            		Save a quick reminder (stored locally)
  history                 		Show command history
  help                    		Show this help
  weather <location?>     		Get weather
  convert|currency <args> 		Currency / unit conversions
  news <args>             		Fetch latest news
  notify <args>           		Send a notification
  alarm|timer <args>      		Schedule alarm/timer (if message channel available)
  speedtest <args>        		Run internet speedtest (non-blocking; streamed output if available)
  ls                      		List directory contents
  calc <expression>       		Calculator
  goal <args>             		Goal tracking helper
  focus <args>            		Start focus session (non-blocking; results/updates streamed)
  create folder <name>     		Create a new folder (alias: mkdir)
  remove folder <name>     		Delete a folder (alias: rmdir)
  delete <file>            		Delete a file (alias: del)
  copy <src> <dst>         		Copy file or folder (alias: cp)
  move <src> <dst>         		Move or rename file/folder (alias: mv)
  read <file>         		Display contents of a file (alias: cat)
  findin <pattern> <file>  	Search inside files (alias: grep)
  tasks|processes          		Show running processes (alias: tasklist)
  kill|end <pid|name>      		Terminate a process (alias: taskkill)
  drives|volumes           		Show connected drives / volumes (alias: get-volume)
  new file <name>         		Create new file
  save <filename>				Save file
`
}

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

func (e *Engine) CmdPwd() string {
	return filepath.Clean(e.cwd)
}

func (e *Engine) CmdCd(args []string) string {
	target := ""
	if len(args) == 0 || args[0] == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "cd: cannot determine home directory"
		}
		target = home
	} else {
		target = args[0]
	}

	if strings.HasPrefix(target, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "cd: cannot determine home directory"
		}
		if target == "~" {
			target = home
		} else if strings.HasPrefix(target, "~/") || strings.HasPrefix(target, `~\`) {
			target = filepath.Join(home, target[2:])
		}
	}

	if !filepath.IsAbs(target) {
		target = filepath.Join(e.cwd, target)
	}

	target = filepath.Clean(target)

	info, err := os.Stat(target)
	if err != nil {
		return fmt.Sprintf("cd: %s: %v", target, err)
	}
	if !info.IsDir() {
		return fmt.Sprintf("cd: %s: not a directory", target)
	}

	if err := os.Chdir(target); err != nil {
		return fmt.Sprintf("cd: failed to change directory: %v", err)
	}

	e.cwd = target
	return ""
}
