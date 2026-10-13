package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ale/metro/internal/audio"
	"github.com/ale/metro/internal/engine"
	"github.com/ale/metro/internal/midi"
	"github.com/ale/metro/internal/store"
	"github.com/ale/metro/internal/ui"
	"github.com/ale/metro/internal/version"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-version", "--version", "-V":
			fmt.Println(version.String())
			return
		}
	}
	st, err := store.Default()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config dir: %v\n", err)
		os.Exit(1)
	}

	player, err := audio.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "audio: %v\n", err)
		os.Exit(1)
	}
	defer player.Close()

	eng := engine.New(player)
	mid := midi.New()
	defer mid.Stop()

	model, err := ui.New(st, eng, player, mid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ui: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "metro: %v\n", err)
		os.Exit(1)
	}
}
