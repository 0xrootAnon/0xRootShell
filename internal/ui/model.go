package ui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/0xrootAnon/0xRootShell/internal/engine"
	"github.com/0xrootAnon/0xRootShell/internal/sound"
	"github.com/0xrootAnon/0xRootShell/internal/store"
)

var (
	artStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#68FF6B")).Bold(true)
	outputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A8FF60"))
	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#B2FF9E"))
	footerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6BFFB8")).Italic(true)
)

// messages for animation ticks
type tickMsg time.Time

type bootDoneMsg struct{}
type printDoneMsg struct{}

type Model struct {
	ascii     string
	input     textinput.Model
	outputBuf []string
	store     *store.Store
	lastExec  time.Time
	engine    *engine.Engine
	width     int
	height    int

	// sound manager (may be nil — audio optional)
	sound *sound.SoundManager

	//boot animation state
	booting       bool
	bootLines     []string //lines to reveal at startup
	bootLineIndex int
	bootCharIndex int

	//command/output typing state
	printing       bool
	printLines     []string //lines being printed for the current command
	printLineIndex int
	printCharIndex int
}

// NewModel now accepts an optional sound manager (pass nil to disable sounds)
func NewModel(st *store.Store, ascii string, sm *sound.SoundManager) Model {
	ti := textinput.New()
	ti.Placeholder = "type a command — e.g. launch chrome, find resume, sys status"
	ti.Focus()
	ti.CharLimit = 512
	ti.Width = 70

	//boot cinematic messages (make tweak here later) (can load from assets too)
	bootMsgs := []string{
		"Loading modules...",
		"Initializing workspace...",
		"Scanning environment...",
		"Mounting cinematic overlays...",
		"Ready.",
	}

	m := Model{
		ascii:     ascii,
		input:     ti,
		store:     st,
		engine:    engine.NewEngine(st),
		booting:   true,
		bootLines: bootMsgs,
		sound:     sm,
	}
	return m
}

func (m Model) Init() tea.Cmd {
	//start the boot tick loop
	return bootTickCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		//handle animation tick
		return m.handleTick()
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		//while booting or printing, ignore keys (i must still allow ctrl+c/esc, haha rich ux mf)
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			//only accept enter when not booting/printing
			if m.booting || m.printing {
				//ignore enter while animating
				return m, nil
			}
			val := strings.TrimSpace(m.input.Value())
			if val != "" {
				//save to history (handle error silently but show)
				if err := m.store.SaveHistory(val); err != nil {
					m.outputBuf = append(m.outputBuf, "history save error: "+err.Error())
				}
				//dispatch command (non-blocking; we capture output and animate)
				rawOut := m.engine.Execute(val)
				//queue for animated printing
				lines := strings.Split(rawOut, "\n")
				//prepare printing state
				m.printLines = lines
				m.printLineIndex = 0
				m.printCharIndex = 0
				m.printing = true
				//push a blank line placeholder to outputBuf for the first printing line
				m.outputBuf = append(m.outputBuf, fmt.Sprintf("> %s", val))
				m.outputBuf = append(m.outputBuf, "") //placeholder for animated output
				m.input.SetValue("")
				//play a small response tone (command accepted)
				if m.sound != nil {
					m.sound.PlayEvent("response")
				}
				//start printing ticks
				return m, printTickCmd()
			}
		}
	}

	//allow text input to process typing keys only if not booting/printing
	if !m.booting && !m.printing {
		m.input, cmd = m.input.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	sb := &strings.Builder{}

	//center the ascii art using the current width
	art := centerArt(m.ascii, m.width)
	sb.WriteString(artStyle.Render(art))
	sb.WriteString("\n")

	//if booting, show the progress lines we have revealed (they're already in outputBuf)
	//else show normal output buffer (keeps latest entries)
	maxLines := 18
	start := 0
	if len(m.outputBuf) > maxLines {
		start = len(m.outputBuf) - maxLines
	}
	for _, line := range m.outputBuf[start:] {
		sb.WriteString(line + "\n")
	}

	//prompt (disabled look when booting/printing)
	if m.booting || m.printing {
		sb.WriteString("\n" + promptStyle.Render("> ") + "(initializing...)" + "\n\n")
	} else {
		sb.WriteString("\n" + promptStyle.Render("> ") + m.input.View() + "\n\n")
	}

	sb.WriteString(footerStyle.Render("0xRootShell — type 'help' — press ESC or Ctrl+C to quit"))
	return sb.String()
}

/***centers each ASCII line horizontally for neat display***/
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

/***animation tick helpers***/

func bootTickCmd() tea.Cmd {
	//tweak boot tick delay here (per character)
	return tea.Tick(30*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func printTickCmd() tea.Cmd {
	//tweak print tick delay here (per character) for cinematic feel
	return tea.Tick(18*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) handleTick() (tea.Model, tea.Cmd) {
	//if booting: reveal characters for current boot line
	if m.booting {
		//if boot finished
		if m.bootLineIndex >= len(m.bootLines) {
			//boot done
			m.booting = false
			//append a blank line to separate boot from regular output
			m.outputBuf = append(m.outputBuf, "")
			// play startup finished sound
			if m.sound != nil {
				m.sound.PlayEvent("startup")
			}
			return m, nil
		}
		line := m.bootLines[m.bootLineIndex]
		runes := []rune(line)
		//if starting this line, append placeholder
		if m.bootCharIndex == 0 {
			m.outputBuf = append(m.outputBuf, "") //placeholder
		}
		//append next rune to the last line (in outputBuf)
		if len(m.outputBuf) == 0 {
			m.outputBuf = append(m.outputBuf, "")
		}
		lastIdx := len(m.outputBuf) - 1
		cur := []rune(m.outputBuf[lastIdx])
		if m.bootCharIndex < len(runes) {
			cur = append(cur, runes[m.bootCharIndex])
			m.outputBuf[lastIdx] = string(cur)
			// play a small typeclick for cinematic feel every 2 chars
			if m.sound != nil && (m.bootCharIndex%2 == 0) {
				m.sound.PlayEvent("typeclick")
			}
			m.bootCharIndex++
			//schedule next tick
			return m, bootTickCmd()
		}
		//line finished, move to next boot line after a short pause
		m.bootLineIndex++
		m.bootCharIndex = 0
		//small pause between lines
		return m, tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
	}

	//if printing command output
	if m.printing {
		//safety: if no lines, end printing
		if len(m.printLines) == 0 {
			m.printing = false
			return m, nil
		}
		//ensure we have a placeholder line in outputBuf for this printing line
		if m.printLineIndex >= len(m.printLines) {
			//done printing all lines
			m.printing = false
			// play small response/done chime
			if m.sound != nil {
				m.sound.PlayEvent("response")
			}
			return m, nil
		}

		currLine := m.printLines[m.printLineIndex]
		runes := []rune(currLine)
		//find the index in outputBuf where the animated printed line sits.
		placeholderIdx := -1
		for i := len(m.outputBuf) - 1; i >= 0; i-- {
			placeholderIdx = i
			break
		}
		if placeholderIdx < 0 {
			//fallback: append one
			m.outputBuf = append(m.outputBuf, "")
			placeholderIdx = len(m.outputBuf) - 1
		}
		//current printed runes for this line:
		curRunes := []rune(m.outputBuf[placeholderIdx])
		if m.printCharIndex < len(runes) {
			curRunes = append(curRunes, runes[m.printCharIndex])
			m.outputBuf[placeholderIdx] = string(curRunes)
			// play typeclick every 2 chars to avoid too many sounds
			if m.sound != nil && (m.printCharIndex%2 == 0) {
				m.sound.PlayEvent("typeclick")
			}
			m.printCharIndex++
			// next char
			return m, printTickCmd()
		}
		//finished this line, move to next line after a short pause
		m.printLineIndex++
		m.printCharIndex = 0
		//if there are more lines, append a new placeholder
		if m.printLineIndex < len(m.printLines) {
			m.outputBuf = append(m.outputBuf, "")
			return m, tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
		}
		//all printed
		m.printing = false
		// final response chime
		if m.sound != nil {
			m.sound.PlayEvent("response")
		}
		return m, nil
	}

	//nothing to do
	return m, nil
}
