package ui

import (
	"fmt"
	"regexp"
	"strconv"
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

type tickMsg time.Time

type bootDoneMsg struct{}
type printDoneMsg struct{}

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

	booting       bool
	bootLines     []string
	bootLineIndex int
	bootCharIndex int

	printing       bool
	printLines     []string
	printLineIndex int
	printCharIndex int

	printPlaceholderIdx int

	asyncCh chan string

	passwordMode       bool
	passwordTargetIdx  int
	passwordTargetSSID string
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
	return tea.Batch(bootTickCmd(), listenCmd(m.asyncCh))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		return m.handleTick()
	case asyncMsg:
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
				return m, nil
			}

			if m.passwordMode {
				pwd := strings.TrimSpace(m.input.Value())
				m.input.SetValue("")
				if setter, ok := interface{}(&m.input).(interface {
					SetEchoMode(int)
					SetEchoCharacter(rune)
				}); ok {
					setter.SetEchoMode(0)
					setter.SetEchoCharacter('*')
				}

				m.passwordMode = false

				cmdline := fmt.Sprintf("net wifi connect %d %s", m.passwordTargetIdx+1, pwd)
				if err := m.store.SaveHistory(cmdline); err != nil {
					m.outputBuf = append(m.outputBuf, "history save error: "+err.Error())
				}
				rawOut := m.engine.Execute(cmdline)

				lines := strings.Split(rawOut, "\n")
				m.printLines = lines
				m.printLineIndex = 0
				m.printCharIndex = 0
				m.printing = true
				m.outputBuf = append(m.outputBuf, fmt.Sprintf("> %s", cmdline))
				m.outputBuf = append(m.outputBuf, "")
				m.printPlaceholderIdx = len(m.outputBuf) - 1
				return m, printTickCmd()
			}

			val := strings.TrimSpace(m.input.Value())
			if val != "" {
				if err := m.store.SaveHistory(val); err != nil {
					m.outputBuf = append(m.outputBuf, "history save error: "+err.Error())
				}
				rawOut := m.engine.Execute(val)

				if strings.HasPrefix(rawOut, "PROMPT_PASSWORD:") {
					parts := strings.SplitN(rawOut[len("PROMPT_PASSWORD:"):], ":", 2)
					if len(parts) == 2 {
						i, _ := strconv.Atoi(parts[0])
						m.passwordTargetIdx = i
						m.passwordTargetSSID = parts[1]
						m.passwordMode = true
						m.input.SetValue("")

						if setter, ok := interface{}(&m.input).(interface {
							SetEchoMode(int)
							SetEchoCharacter(rune)
						}); ok {
							setter.SetEchoMode(1)
							setter.SetEchoCharacter('*')
						}
						m.outputBuf = append(m.outputBuf, fmt.Sprintf("> %s", val))
						m.outputBuf = append(m.outputBuf, fmt.Sprintf("(enter password for '%s')", m.passwordTargetSSID))
						return m, nil
					}
				}

				lines := strings.Split(rawOut, "\n")
				m.printLines = lines
				m.printLineIndex = 0
				m.printCharIndex = 0
				m.printing = true
				m.outputBuf = append(m.outputBuf, fmt.Sprintf("> %s", val))
				m.outputBuf = append(m.outputBuf, "")
				m.printPlaceholderIdx = len(m.outputBuf) - 1
				m.input.SetValue("")
				return m, printTickCmd()
			}
		}
	}

	if !m.booting && !m.printing {
		m.input, cmd = m.input.Update(msg)
	}

	return m, cmd
}

func (m Model) View() string {
	sb := &strings.Builder{}

	art := centerArt(m.ascii, m.width)
	sb.WriteString(artStyle.Render(art))
	sb.WriteString("\n")

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
	if m.booting {
		if m.bootLineIndex >= len(m.bootLines) {
			m.booting = false
			m.outputBuf = append(m.outputBuf, "")
			return m, nil
		}
		line := m.bootLines[m.bootLineIndex]
		runes := []rune(line)
		if m.bootCharIndex == 0 {
			m.outputBuf = append(m.outputBuf, "")
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

	if m.printing {
		if len(m.printLines) == 0 {
			m.printing = false
			m.printPlaceholderIdx = -1
			return m, nil
		}
		if m.printLineIndex >= len(m.printLines) {
			m.printing = false
			m.printPlaceholderIdx = -1
			return m, nil
		}

		currLine := m.printLines[m.printLineIndex]
		runes := []rune(currLine)

		placeholderIdx := m.printPlaceholderIdx + m.printLineIndex
		for placeholderIdx >= len(m.outputBuf) {
			m.outputBuf = append(m.outputBuf, "")
		}
		curRunes := []rune(m.outputBuf[placeholderIdx])
		if m.printCharIndex < len(runes) {
			curRunes = append(curRunes, runes[m.printCharIndex])
			m.outputBuf[placeholderIdx] = string(curRunes)
			m.printCharIndex++
			return m, printTickCmd()
		}
		m.printLineIndex++
		m.printCharIndex = 0
		if m.printLineIndex < len(m.printLines) {
			m.outputBuf = append(m.outputBuf, "")
			return m, tea.Tick(120*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
		}
		m.printing = false
		m.printPlaceholderIdx = -1
		return m, nil
	}

	return m, nil
}
