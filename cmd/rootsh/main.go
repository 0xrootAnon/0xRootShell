package main

import (
	_ "embed"
	"log"
	"os"

	"github.com/0xrootAnon/0xRootShell/internal/store"
	"github.com/0xrootAnon/0xRootShell/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

//go:embed assets/ascii.txt
var embeddedAscii string

func main() {
	// ensure data directory exists
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		_ = os.Mkdir("data", 0755)
	}

	// use embedded ascii as default, allow on-disk override
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
