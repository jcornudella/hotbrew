package cmd

import "fmt"

func (r *Root) cmdVersion(args []string) error {
	fmt.Printf("hotbrew v%s\n", r.version)
	return nil
}

func (r *Root) cmdHelp(args []string) error {
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
	return nil
}
