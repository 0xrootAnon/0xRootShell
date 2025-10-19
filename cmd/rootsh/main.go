package main

import (
	"log"
	"os"

	"github.com/0xrootAnon/0xRootShell/internal/store"
	"github.com/0xrootAnon/0xRootShell/internal/ui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	//ensure data directory
	if _, err := os.Stat("data"); os.IsNotExist(err) {
		_ = os.Mkdir("data", 0755)
	}

	//lets load ascii art from assets
	asciiArt := ""
	if b, err := os.ReadFile("assets/ascii.txt"); err == nil {
		asciiArt = string(b)
	} else {
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
