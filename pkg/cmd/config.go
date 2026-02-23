package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/ui/theme"
	"github.com/jcornudella/hotbrew/pkg/profile"
)

func (r *Root) cmdConfig(args []string) error {
	if len(args) > 0 && args[0] == "--init" {
		if err := config.Init(); err != nil {
			return fmt.Errorf("create config: %w", err)
		}
		fmt.Println("Config created at ~/.config/hotbrew/hotbrew.yaml")
		return nil
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "hotbrew", "hotbrew.yaml")

	if err := config.Init(); err != nil {
		return fmt.Errorf("create config: %w", err)
	}

	fmt.Printf("Config file: %s\n", configPath)
	fmt.Printf("Open with: %s %s\n", editor, configPath)
	return nil
}

func (r *Root) cmdTheme(args []string) error {
	available := theme.List()
	if len(args) == 0 {
		fmt.Println("☕ Available themes:")
		for _, t := range available {
			fmt.Printf("  • %s\n", t)
		}
		fmt.Println("\nSet theme with: hotbrew theme <name>")
		return nil
	}

	requested := args[0]
	if !contains(available, requested) {
		fmt.Printf("Unknown theme '%s'. Run 'hotbrew theme' to list options.\n", requested)
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	cfg.Theme = requested
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Printf("✓ Theme switched to %s\n", requested)
	fmt.Println("Restart hotbrew to apply changes.")
	return nil
}

func contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}

func (r *Root) cmdLogin(args []string) error {
	if len(args) < 1 {
		fmt.Println("Usage: hotbrew login <token>")
		fmt.Println("\nGet your token at https://hotbrew.dev")
		return nil
	}

	token := args[0]

	home, _ := os.UserHomeDir()
	tokenPath := filepath.Join(home, ".config", "hotbrew", "token")

	if err := os.MkdirAll(filepath.Dir(tokenPath), 0755); err != nil {
		return fmt.Errorf("create token dir: %w", err)
	}
	if err := os.WriteFile(tokenPath, []byte(token), 0600); err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	fmt.Println("☕ Logged in successfully!")
	fmt.Println("")
	fmt.Println("Next steps:")
	fmt.Println("  1. Run 'hotbrew' to see your newsletter")
	fmt.Println("  2. Run 'hotbrew setup' to add hotbrew to your shell")
	fmt.Println("")
	return nil
}

func (r *Root) cmdSetup(args []string) error {
	shell := os.Getenv("SHELL")
	var rcFile string

	switch {
	case strings.Contains(shell, "zsh"):
		rcFile = "~/.zshrc"
	case strings.Contains(shell, "bash"):
		rcFile = "~/.bashrc"
	case strings.Contains(shell, "fish"):
		rcFile = "~/.config/fish/config.fish"
	default:
		rcFile = "your shell's rc file"
	}

	fmt.Println("☕ Setup hotbrew to run on terminal open")
	fmt.Println("")
	fmt.Printf("Add this line to %s:\n", rcFile)
	fmt.Println("")
	fmt.Println("  # Show hotbrew on new terminal")
	fmt.Println("  hotbrew")
	fmt.Println("")
	fmt.Println("Or for a less intrusive option:")
	fmt.Println("")
	fmt.Println("  # Show hotbrew greeting")
	fmt.Println("  echo \"☕ Run 'hotbrew' for your morning digest\"")
	fmt.Println("")
	return nil
}

func isFirstRun() bool {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "hotbrew", "hotbrew.yaml")
	_, err := os.Stat(configPath)
	return os.IsNotExist(err)
}

func runFirstTimeSetup() error {
	fmt.Println("")
	fmt.Println("    \033[38;5;117m) )\033[0m")
	fmt.Println("   \033[38;5;117m( (\033[0m")
	fmt.Println("    \033[38;5;117m) )\033[0m")
	fmt.Println("   \033[38;5;205m______\033[0m")
	fmt.Println("  \033[38;5;205m|      |]\033[0m")
	fmt.Println("  \033[38;5;205m|      |\033[0m")
	fmt.Println("   \033[38;5;205m\\____/\033[0m")
	fmt.Println("")
	fmt.Println("\033[1m☕ Welcome to hotbrew!\033[0m")
	fmt.Println("")
	fmt.Println("Let's get you set up in 10 seconds...")
	fmt.Println("")

	if err := config.Init(); err != nil {
		return err
	}
	_ = profile.EnsureDefaultProfile()

	fmt.Println("Pick a theme:")
	fmt.Println("  [1] synthwave - Neon pink/purple (default)")
	fmt.Println("  [2] mocha     - Coffee browns")
	fmt.Println("  [3] nord      - Arctic blues")
	fmt.Println("  [4] dracula   - Dark purples")
	fmt.Println("  [5] ocean     - Deep sea")
	fmt.Print("\nChoice [1-5, or Enter for default]: ")

	var choice string
	fmt.Scanln(&choice)

	themes := map[string]string{
		"1": "synthwave", "2": "mocha", "3": "nord",
		"4": "dracula", "5": "ocean", "": "synthwave",
	}

	selectedTheme := themes[choice]
	if selectedTheme == "" {
		selectedTheme = "synthwave"
	}

	cfg, _ := config.Load()
	cfg.Theme = selectedTheme
	config.Save(cfg)

	fmt.Printf("\n✓ Theme set to %s\n", selectedTheme)

	fmt.Print("\nAdd hotbrew to your shell (shows on terminal open)? [Y/n]: ")
	var addShell string
	fmt.Scanln(&addShell)

	if addShell == "" || strings.ToLower(addShell) == "y" {
		shell := os.Getenv("SHELL")
		home, _ := os.UserHomeDir()
		var rcFile string

		switch {
		case strings.Contains(shell, "zsh"):
			rcFile = filepath.Join(home, ".zshrc")
		case strings.Contains(shell, "bash"):
			rcFile = filepath.Join(home, ".bashrc")
		}

		if rcFile != "" {
			f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				f.WriteString("\n# hotbrew - Your morning, piping hot\n")
				f.WriteString("command -v hotbrew &>/dev/null && hotbrew\n")
				f.Close()
				fmt.Printf("✓ Added to %s\n", rcFile)
			}
		}
	}

	fmt.Println("")
	fmt.Println("\033[1m☕ You're all set! Loading your first brew...\033[0m")
	fmt.Println("")
	return nil
}
