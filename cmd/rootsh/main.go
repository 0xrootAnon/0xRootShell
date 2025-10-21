package main

import (
	_ "embed"
	"io"
	"log"
	"os"

	"github.com/0xrootAnon/0xRootShell/internal/store"
	"github.com/0xrootAnon/0xRootShell/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

//go:embed assets/ascii.txt
var embeddedAscii string

func main() {
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		_ = os.Mkdir("data", 0755)
	}
	_ = os.MkdirAll("data/screenshots", 0755)
	_ = os.MkdirAll("data/recordings", 0755)
	_ = os.MkdirAll("data/outgoing_messages", 0755)
	_ = os.MkdirAll("data/cache", 0755)

	if os.Getenv("DEBUG") != "" {
		f, err := os.OpenFile("data/debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			log.SetOutput(io.MultiWriter(os.Stderr, f))
			log.SetFlags(log.LstdFlags | log.Lshortfile)
			log.Println("DEBUG mode enabled")
		} else {
			log.SetOutput(os.Stderr)
			log.Println("DEBUG: could not open data/debug.log:", err)
		}
	}

	asciiArt := embeddedAscii
	if b, err := os.ReadFile("../assets/ascii.txt"); err == nil && len(b) > 0 {
		asciiArt = string(b)
	} else if asciiArt == "" {
		asciiArt = "0xRootShell"
	}

	dbPath := "data/0xrootshell.db"
	st, err := store.NewStore(dbPath)
	if err != nil {
		log.Fatalf("store init: %v", err)
	}
	defer st.Close()

	m := ui.NewModel(st, asciiArt)

	prog := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		log.Fatalf("program failed: %v", err)
	}
}
