package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"curler/internal/app"
	"curler/internal/config"
)

func main() {
	urlFlag := flag.String("url", "", "Directly add a URL to download queue on launch")
	saveDirFlag := flag.String("dir", "", "Custom save directory")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Println("curler v1.0.0 (Arch Linux Edition)")
		curlVer, err := config.CheckCurlAvailable()
		if err != nil {
			fmt.Printf("curl status: NOT FOUND (%v)\n", err)
		} else {
			fmt.Printf("curl status: %s\n", curlVer)
		}
		os.Exit(0)
	}

	// Ensure curl binary exists
	_, err := config.CheckCurlAvailable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: 'curl' binary is required but was not found in PATH.\nPlease install curl using 'sudo pacman -S curl'.\n")
		os.Exit(1)
	}

	cfg := config.DefaultConfig()
	if *saveDirFlag != "" {
		cfg.DownloadDir = *saveDirFlag
	}

	model := app.NewModel(cfg)

	// If direct URL was passed via CLI flag, enqueue it immediately
	if *urlFlag != "" {
		_, _ = model.Manager.AddTask(*urlFlag, "", cfg.DownloadDir)
	}

	// Create Bubble Tea program with Mouse cell motion and AltScreen enabled
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Background ticker to dispatch UI update frames
	go func() {
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for range ticker.C {
			p.Send(app.UpdateStateMsg{})
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running curler TUI: %v\n", err)
		os.Exit(1)
	}

	// Save final state before exiting
	model.Manager.SaveState()
}
