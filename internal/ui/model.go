package ui

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/0xrootAnon/0xRootShell/internal/engine"
	"github.com/0xrootAnon/0xRootShell/internal/store"
)

var (
	artStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#68FF6B")).Bold(true)
	outputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A8FF60"))
	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#B2FF9E"))
	footerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6BFFB8")).Italic(true)
	uiAnsiRe    = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
)

// messages for animation ticks
type tickMsg time.Time

type bootDoneMsg struct{}
type printDoneMsg struct{}

// asyncMsg is used when the engine sends asynchronous strings (e.g., find results)
type asyncMsg string

type Model struct {
	ascii     string
	input     textinput.Model
	outputBuf []string
	store     *store.Store
	lastExec  time.Time
	engine    *engine.Engine
	width     int
	height    int

	// boot animation state
	booting       bool
	bootLines     []string // lines to reveal at startup
	bootLineIndex int
	bootCharIndex int

	// command/output typing state
	printing       bool
	printLines     []string // lines being printed for the current command
	printLineIndex int
	printCharIndex int

	// index in outputBuf where the first placeholder for the current printing session is located.
	// when not printing, set to -1.
	printPlaceholderIdx int

	// async channel to receive engine messages (background results)
	asyncCh chan string
}

func sanitizeForUI(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = uiAnsiRe.ReplaceAllString(s, "")
	return s
}

func NewModel(st *store.Store, ascii string) Model {
	ti := textinput.New()
	ti.Placeholder = "type a command — e.g. launch chrome, find resume, sys status"
	ti.Focus()
	ti.CharLimit = 512
	ti.Width = 70

	bootMsgs := []string{
		"Loading modules...",
		"Initializing workspace...",
		"Scanning environment...",
		"Mounting cinematic overlays...",
		"Ready.",
	}

	// create message channel for engine -> UI communication
	ch := make(chan string, 16)

	m := Model{
		ascii:               ascii,
		input:               ti,
		store:               st,
		engine:              engine.NewEngine(st, ch),
		booting:             true,
		bootLines:           bootMsgs,
		printPlaceholderIdx: -1,
		asyncCh:             ch,
	}
	return m
}

func (m Model) Init() tea.Cmd {
	// start both the boot tick and the async listener
	return tea.Batch(bootTickCmd(), listenCmd(m.asyncCh))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		return m.handleTick()
	case asyncMsg:
		// append the async message to output buffer and re-arm the listener
		m.outputBuf = append(m.outputBuf, string(msg))
		return m, listenCmd(m.asyncCh)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			if m.booting || m.printing {
				// ignore enter while animating
				return m, nil
			}
			val := strings.TrimSpace(m.input.Value())
			if val != "" {
				// save to history
				if err := m.store.SaveHistory(val); err != nil {
					m.outputBuf = append(m.outputBuf, "history save error: "+err.Error())
				}
				// dispatch command
				rawOut := m.engine.Execute(val)
				rawOut = sanitizeForUI(rawOut)
				// prepare lines for printing
				lines := strings.Split(rawOut, "\n")
				m.printLines = lines
				m.printLineIndex = 0
				m.printCharIndex = 0
				m.printing = true
				// push the command line
				m.outputBuf = append(m.outputBuf, fmt.Sprintf("> %s", val))
				// append the first placeholder and record its index
				m.outputBuf = append(m.outputBuf, "") // placeholder for first printed output line
				m.printPlaceholderIdx = len(m.outputBuf) - 1
				m.input.SetValue("")
				// start printing ticks
				return m, printTickCmd()
			}
		}
	}

	// allow text input to process typing keys only if not booting/printing
	if !m.booting && !m.printing {
		m.input, cmd = m.input.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	sb := &strings.Builder{}

	// center ascii art
	art := centerArt(m.ascii, m.width)
	sb.WriteString(artStyle.Render(art))
	sb.WriteString("\n")

	// show last lines of outputBuf
	// make visible lines dynamic based on terminal height; keep minimum sensible size
	maxLines := m.height - 8
	if maxLines < 6 {
		maxLines = 6
	}
	start := 0
	if len(m.outputBuf) > maxLines {
		start = len(m.outputBuf) - maxLines
	}
	for _, line := range m.outputBuf[start:] {
		sb.WriteString(line + "\n")
	}

	// prompt (disabled look when booting/printing)
	if m.booting || m.printing {
		sb.WriteString("\n" + promptStyle.Render("> ") + "(initializing...)" + "\n\n")
	} else {
		sb.WriteString("\n" + promptStyle.Render("> ") + m.input.View() + "\n\n")
	}

	sb.WriteString(footerStyle.Render("0xRootShell — type 'help' — press ESC or Ctrl+C to quit"))
	return sb.String()
}

func centerArt(ascii string, width int) string {
	if width <= 0 {
		return ascii
	}
	var b strings.Builder
	for _, line := range strings.Split(ascii, "\n") {
		trim := strings.TrimRight(line, " ")
		rlen := utf8.RuneCountInString(trim)
		padding := 0
		if rlen < width {
			padding = (width - rlen) / 2
		}
		if padding > 0 {
			b.WriteString(strings.Repeat(" ", padding))
		}
		b.WriteString(trim + "\n")
	}
	return b.String()
}

/*** animation tick helpers ***/

func bootTickCmd() tea.Cmd {
	return tea.Tick(30*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func printTickCmd() tea.Cmd {
	return tea.Tick(18*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// listenCmd returns a tea.Cmd which waits for one message on the channel and
// converts it into an asyncMsg for the main Update loop. After receiving one
// message the Update handler re-arms the listener by returning listenCmd again.
func listenCmd(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		s, ok := <-ch
		if !ok {
			return nil
		}
		return asyncMsg(s)
	}
}

func (m Model) handleTick() (tea.Model, tea.Cmd) {
	// boot sequence
	if m.booting {
		if m.bootLineIndex >= len(m.bootLines) {
			m.booting = false
			m.outputBuf = append(m.outputBuf, "")
			return m, nil
		}
		line := m.bootLines[m.bootLineIndex]
		runes := []rune(line)
		if m.bootCharIndex == 0 {
			m.outputBuf = append(m.outputBuf, "") // placeholder
		}
		if len(m.outputBuf) == 0 {
			m.outputBuf = append(m.outputBuf, "")
		}
		lastIdx := len(m.outputBuf) - 1
		cur := []rune(m.outputBuf[lastIdx])
		if m.bootCharIndex < len(runes) {
			cur = append(cur, runes[m.bootCharIndex])
			m.outputBuf[lastIdx] = string(cur)
			m.bootCharIndex++
			return m, bootTickCmd()
		}
		m.bootLineIndex++
		m.bootCharIndex = 0
		return m, tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
	}

	// printing command output
	if m.printing {
		// safety
		if len(m.printLines) == 0 {
			m.printing = false
			m.printPlaceholderIdx = -1
			return m, nil
		}
		// finished all lines
		if m.printLineIndex >= len(m.printLines) {
			m.printing = false
			m.printPlaceholderIdx = -1
			return m, nil
		}

		currLine := m.printLines[m.printLineIndex]
		runes := []rune(currLine)

		// compute placeholder index robustly using recorded base index
		placeholderIdx := m.printPlaceholderIdx + m.printLineIndex
		// safety: if placeholder index is out of bounds, append placeholders until it exists
		for placeholderIdx >= len(m.outputBuf) {
			m.outputBuf = append(m.outputBuf, "")
		}
		// current printed runes for this line:
		curRunes := []rune(m.outputBuf[placeholderIdx])
		if m.printCharIndex < len(runes) {
			curRunes = append(curRunes, runes[m.printCharIndex])
			m.outputBuf[placeholderIdx] = string(curRunes)
			// play typewriter beep (non-blocking)
			m.printCharIndex++
			// schedule next char tick
			return m, printTickCmd()
		}
		// finished this line
		m.printLineIndex++
		m.printCharIndex = 0
		// if there are more lines, append a new placeholder (which keeps printPlaceholderIdx correct)
		if m.printLineIndex < len(m.printLines) {
			m.outputBuf = append(m.outputBuf, "")
			return m, tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
		}
		// all printed
		m.printing = false
		m.printPlaceholderIdx = -1
		return m, nil
	}

	// nothing
	return m, nil
}
