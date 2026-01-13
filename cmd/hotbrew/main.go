package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/ui"
)

var Version = "0.1.0"

func main() {
	// Handle subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "config":
			handleConfig()
			return
		case "themes":
			handleThemes()
			return
		case "help", "--help", "-h":
			handleHelp()
			return
		case "version", "--version", "-v":
			fmt.Printf("hotbrew v%s\n", Version)
			return
		}
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Run the app
	model := ui.NewModel(cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleConfig() {
	if len(os.Args) > 2 && os.Args[2] == "--init" {
		if err := config.Init(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Config created at ~/.config/hotbrew/hotbrew.yaml")
		return
	}

	// Open config in editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	home, _ := os.UserHomeDir()
	configPath := home + "/.config/hotbrew/hotbrew.yaml"

	// Ensure config exists
	if err := config.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Config file: %s\n", configPath)
	fmt.Printf("Open with: %s %s\n", editor, configPath)
}

func handleThemes() {
	themes := []string{"synthwave", "nord", "dracula"}
	fmt.Println("☕ Available themes:")
	for _, t := range themes {
		fmt.Printf("  • %s\n", t)
	}
	fmt.Println("\nSet theme in ~/.config/hotbrew/hotbrew.yaml")
}

func handleHelp() {
	help := `
☕ hotbrew - Your morning, piping hot

USAGE:
    hotbrew              Show your digest
    hotbrew config       View config file location
    hotbrew config --init    Create default config
    hotbrew themes       List available themes
    hotbrew help         Show this help

KEYBOARD SHORTCUTS:
    j/k, ↑/↓    Navigate items
    tab         Next section
    enter, e    Expand/collapse item
    o           Open in browser
    c           Open comments (Hacker News)
    r           Refresh
    1-9         Jump to section
    q, esc      Quit

CONFIGURATION:
    Config file: ~/.config/hotbrew/hotbrew.yaml

    Example config:
        theme: synthwave
        sources:
          hackernews:
            enabled: true
            settings:
              max: 10

For more info: https://github.com/jcornudella/hotbrew
`
	fmt.Print(help)
}
