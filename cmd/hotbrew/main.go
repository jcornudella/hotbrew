package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/sources/arxiv"
	"github.com/jcornudella/hotbrew/internal/sources/github"
	"github.com/jcornudella/hotbrew/internal/sources/hackernews"
	"github.com/jcornudella/hotbrew/internal/sources/hnsearch"
	"github.com/jcornudella/hotbrew/internal/sources/lobsters"
	"github.com/jcornudella/hotbrew/internal/sources/reddit"
	"github.com/jcornudella/hotbrew/internal/sources/tldr"
	"github.com/jcornudella/hotbrew/internal/store"
	"github.com/jcornudella/hotbrew/internal/cli"
	"github.com/jcornudella/hotbrew/internal/curation"
	"github.com/jcornudella/hotbrew/internal/daemon"
	hsync "github.com/jcornudella/hotbrew/internal/sync"
	"github.com/jcornudella/hotbrew/internal/ui"
	"github.com/jcornudella/hotbrew/pkg/source"
	"github.com/jcornudella/hotbrew/pkg/trss"
	"github.com/jcornudella/hotbrew/server"
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
		case "login":
			handleLogin()
			return
		case "sync":
			handleSync()
			return
		case "digest":
			handleDigest()
			return
		case "add":
			withStore(func(st *store.Store) { cli.Add(st, os.Args[2:]) })
			return
		case "list", "ls":
			withStore(func(st *store.Store) { handleList(st) })
			return
		case "open":
			id := ""
			if len(os.Args) > 2 {
				id = os.Args[2]
			}
			withStore(func(st *store.Store) { cli.Open(st, id) })
			return
		case "save":
			id := ""
			if len(os.Args) > 2 {
				id = os.Args[2]
			}
			withStore(func(st *store.Store) { cli.Save(st, id) })
			return
		case "mute":
			domain := ""
			if len(os.Args) > 2 {
				domain = os.Args[2]
			}
			withStore(func(st *store.Store) { cli.Mute(st, domain) })
			return
		case "boost":
			tag := ""
			if len(os.Args) > 2 {
				tag = os.Args[2]
			}
			withStore(func(st *store.Store) { cli.Boost(st, tag) })
			return
		case "rules":
			withStore(func(st *store.Store) {
				if len(os.Args) > 2 && os.Args[2] == "--delete" && len(os.Args) > 3 {
					cli.DeleteRule(st, os.Args[3])
				} else {
					cli.Rules(st)
				}
			})
			return
		case "sources":
			withStore(func(st *store.Store) { cli.Sources(st) })
			return
		case "curate":
			withStore(func(st *store.Store) { handleCurate(st) })
			return
		case "stream":
			cfg, _ := config.Load()
			cli.Stream(cfg)
			return
		case "daemon":
			handleDaemon()
			return
		case "serve":
			handleServe()
			return
		case "setup":
			handleSetup()
			return
		case "help", "--help", "-h":
			handleHelp()
			return
		case "version", "--version", "-v":
			fmt.Printf("hotbrew v%s\n", Version)
			return
		}
	}

	// Check for first run
	if isFirstRun() {
		runFirstTimeSetup()
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Try to open store for TUI (graceful — works without it).
	st, _ := store.Open(cfg.GetDBPath())
	if st != nil {
		defer st.Close()
	}

	// Run the app
	model := ui.NewModel(cfg, st)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// withStore opens the store, runs fn, then closes.
func withStore(fn func(st *store.Store)) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	st, err := store.Open(cfg.GetDBPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()
	fn(st)
}

// handleCurate parses curate flags and calls cli.Curate.
func handleCurate(st *store.Store) {
	opts := cli.CurateOptions{}
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--title":
			if i+1 < len(os.Args) {
				i++
				opts.Title = os.Args[i]
			}
		case "--tags":
			if i+1 < len(os.Args) {
				i++
				opts.Tags = strings.Split(os.Args[i], ",")
			}
		case "--note":
			if i+1 < len(os.Args) {
				i++
				opts.Note = os.Args[i]
			}
		default:
			if opts.URL == "" && !strings.HasPrefix(os.Args[i], "-") {
				opts.URL = os.Args[i]
			}
		}
	}
	cli.Curate(st, opts)
}

// handleList parses list flags and calls cli.List.
func handleList(st *store.Store) {
	opts := cli.ListOptions{Top: 20}
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--unread":
			opts.Unread = true
		case "--source":
			if i+1 < len(os.Args) {
				i++
				opts.SourceName = os.Args[i]
			}
		case "--top":
			if i+1 < len(os.Args) {
				i++
				fmt.Sscanf(os.Args[i], "%d", &opts.Top)
			}
		}
	}
	cli.List(st, opts)
}

// handleDaemon dispatches daemon start/stop/status.
func handleDaemon() {
	action := "status"
	if len(os.Args) > 2 {
		action = os.Args[2]
	}

	switch action {
	case "start":
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}
		registry := buildRegistry(cfg)
		if err := daemon.Start(cfg, registry); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "stop":
		if err := daemon.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "status":
		daemon.Status()
	default:
		fmt.Println("Usage: hotbrew daemon [start|stop|status]")
	}
}

func isFirstRun() bool {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "hotbrew", "hotbrew.yaml")
	_, err := os.Stat(configPath)
	return os.IsNotExist(err)
}

func runFirstTimeSetup() {
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

	// Create default config
	if err := config.Init(); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating config: %v\n", err)
	}

	// Ask about theme
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

	// Update config with selected theme
	cfg, _ := config.Load()
	cfg.Theme = selectedTheme
	config.Save(cfg)

	fmt.Printf("\n✓ Theme set to %s\n", selectedTheme)

	// Ask about shell integration
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
	themes := []string{"synthwave", "nord", "dracula", "mocha", "ocean", "forest", "sunset", "midnight"}
	fmt.Println("☕ Available themes:")
	for _, t := range themes {
		fmt.Printf("  • %s\n", t)
	}
	fmt.Println("\nSet theme in ~/.config/hotbrew/hotbrew.yaml")
}

// handleLogin saves the user's token for syncing
func handleLogin() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: hotbrew login <token>")
		fmt.Println("\nGet your token at https://hotbrew.dev")
		os.Exit(1)
	}

	token := os.Args[2]

	// Validate token with server (optional - works offline too)
	// For now just save it

	home, _ := os.UserHomeDir()
	tokenPath := filepath.Join(home, ".config", "hotbrew", "token")

	os.MkdirAll(filepath.Dir(tokenPath), 0755)
	if err := os.WriteFile(tokenPath, []byte(token), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving token: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("☕ Logged in successfully!")
	fmt.Println("")
	fmt.Println("Next steps:")
	fmt.Println("  1. Run 'hotbrew' to see your newsletter")
	fmt.Println("  2. Run 'hotbrew setup' to add hotbrew to your shell")
	fmt.Println("")
}

// handleDigest generates a curated digest from stored items.
func handleDigest() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	st, err := store.Open(cfg.GetDBPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	engine := curation.NewEngine(st)
	window := cfg.GetDigestWindow()
	maxItems := cfg.GetDigestMax()

	digest, err := engine.GenerateDigest(window, maxItems, "Hotbrew Digest")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating digest: %v\n", err)
		os.Exit(1)
	}

	// Check for --json flag
	if len(os.Args) > 2 && os.Args[2] == "--json" {
		trss.EncodeDigest(os.Stdout, digest)
		return
	}

	// Pretty-print the digest
	printDigest(digest)
}

// printDigest renders a digest to the terminal.
func printDigest(d *trss.Digest) {
	fmt.Printf("\n☕ %s\n", d.Title)
	fmt.Printf("   %s | %d items | %d sources\n\n",
		d.GeneratedAt.Local().Format("Jan 2, 3:04 PM"),
		d.ItemCount, d.Meta.SourcesSynced)

	for i, item := range d.Items {
		score := "  "
		if item.Score >= 7 {
			score = "🔥"
		} else if item.Score >= 4 {
			score = "⭐"
		}

		fmt.Printf("  %s %2d. %s\n", score, i+1, item.Title)
		if item.Summary != "" {
			summary := item.Summary
			if len(summary) > 120 {
				summary = summary[:117] + "..."
			}
			fmt.Printf("       %s\n", summary)
		}
		fmt.Printf("       %s %s", item.Source.Icon, item.Source.Name)
		if item.URL != "" {
			fmt.Printf(" · %s", item.URL)
		}
		fmt.Println()
		fmt.Println()
	}

	if d.Meta.ItemsDeduped > 0 || d.Meta.RulesApplied > 0 {
		fmt.Printf("  --- %d deduped, %d rules applied ---\n\n",
			d.Meta.ItemsDeduped, d.Meta.RulesApplied)
	}
}

// handleSync fetches all sources and stores items in SQLite.
// Use --remote to sync config from the server instead.
func handleSync() {
	if len(os.Args) > 2 && os.Args[2] == "--remote" {
		handleSyncRemote()
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	st, err := store.Open(cfg.GetDBPath())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening store: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	registry := buildRegistry(cfg)

	fmt.Println("☕ Syncing sources...")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	results := hsync.SyncAll(ctx, st, registry)
	hsync.PrintResults(results)

	fmt.Printf("\nTotal items in store: %d\n", st.ItemCount())
}

// buildRegistry creates a source.Registry from the config.
func buildRegistry(cfg *config.Config) *source.Registry {
	registry := source.NewRegistry()

	// Hacker News top stories
	if src, ok := cfg.Sources["hackernews"]; ok && src.Enabled {
		registry.Register("hackernews", hackernews.New())
	}

	// Claude Code & Vibe Coding HN search
	registry.Register("hnsearch-claude", hnsearch.New(
		"Claude Code & Vibe Coding",
		[]string{"Claude Code", "vibe coding", "AI coding assistant", "Anthropic Claude"},
		"🤖",
	))

	// GitHub trending AI repos
	registry.Register("github-trending", github.New(
		"GitHub Trending",
		[]string{"ai", "llm", "machine-learning", "gpt", "claude"},
		"🐙",
	))

	// TLDR newsletters
	registry.Register("tldr-ai", tldr.NewAI())
	registry.Register("tldr-tech", tldr.NewTech())

	// Lobste.rs (AI/ML/programming)
	registry.Register("lobsters", lobsters.New(
		"Lobste.rs",
		[]string{"ai", "ml", "programming", "compsci", "plt"},
		"🦞",
	))

	// Reddit AI/ML subreddits
	registry.Register("reddit-ai", reddit.New(
		"Reddit AI",
		[]string{"MachineLearning", "LocalLLaMA", "ClaudeAI"},
		"🔮",
	))

	// arXiv LLM research (mirroring llm-research-digest categories)
	registry.Register("arxiv-llm", arxiv.New(
		"LLM Research",
		arxiv.DefaultCategories,
		"📄",
	))

	return registry
}

// handleSyncRemote syncs config from the remote server.
func handleSyncRemote() {
	home, _ := os.UserHomeDir()
	tokenPath := filepath.Join(home, ".config", "hotbrew", "token")

	tokenData, err := os.ReadFile(tokenPath)
	if err != nil {
		fmt.Println("Not logged in. Run 'hotbrew login <token>' first.")
		fmt.Println("Get your token at https://hotbrew.dev")
		os.Exit(1)
	}

	token := strings.TrimSpace(string(tokenData))

	serverURL := os.Getenv("HOTBREW_SERVER")
	if serverURL == "" {
		serverURL = "https://hotbrew.dev"
	}

	resp, err := http.Get(serverURL + "/api/config/" + token)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		fmt.Println("Invalid token. Please check your token or subscribe at https://hotbrew.dev")
		os.Exit(1)
	}

	var remoteCfg map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&remoteCfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("☕ Remote config synced!")
	fmt.Printf("Theme: %v\n", remoteCfg["theme"])
}

// handleServe starts the hotbrew server
func handleServe() {
	addr := ":8080"
	if len(os.Args) > 2 {
		addr = os.Args[2]
	}

	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".config", "hotbrew", "server")

	fmt.Println("☕ Starting hotbrew server...")
	if err := server.Run(addr, dataDir); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

// handleSetup helps users add hotbrew to their shell
func handleSetup() {
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
}

func handleHelp() {
	help := `
☕ hotbrew — Terminal RSS, piping hot

USAGE:
    hotbrew                  Launch TUI digest viewer
    hotbrew sync             Fetch all sources → SQLite
    hotbrew digest           Show curated digest (pretty)
    hotbrew digest --json    Output as TRSS NDJSON
    hotbrew list [flags]     List items from store
    hotbrew open <id>        Open item in browser, mark read
    hotbrew save <id>        Save an item for later
    hotbrew add <url> [name] Add an RSS feed source
    hotbrew sources          List registered sources
    hotbrew curate <url>     Manually save a link (auto-fetches title)
    hotbrew mute <domain>    Mute a domain
    hotbrew boost <tag>      Boost items with a tag
    hotbrew rules            List active rules
    hotbrew stream           Tail the stream log
    hotbrew daemon start     Start background sync daemon
    hotbrew daemon stop      Stop the daemon
    hotbrew daemon status    Check daemon status
    hotbrew config           View config file location
    hotbrew themes           List available themes
    hotbrew setup            Shell integration instructions
    hotbrew serve [addr]     Run the web server
    hotbrew help             Show this help

LIST FLAGS:
    --unread            Only show unread items
    --source <name>     Filter by source name
    --top <n>           Show top N items (default 20)

TUI SHORTCUTS:
    j/k, ↑/↓    Navigate items
    tab          Next section
    enter, e     Expand/collapse item
    o            Open in browser
    s            Save item
    u            Toggle read/unread
    m            Mute item's domain
    c            Open comments (HN)
    r            Refresh
    1-9          Jump to section
    q, esc       Quit

THEMES:
    synthwave, nord, dracula, mocha, ocean, forest, sunset, midnight

QUICK START:
    hotbrew sync && hotbrew digest

For more info: https://github.com/jcornudella/hotbrew
`
	fmt.Print(help)
}
