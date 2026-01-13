# Digest

Your personalized terminal newsletter. A beautiful CLI tool that aggregates information from multiple sources into a single, scannable digest.

![Digest Demo](docs/demo.gif)

## Features

- **Beautiful UI** - Gradient text, themed colors, and smooth interactions
- **Multiple themes** - Synthwave (default), Nord, Dracula
- **Pluggable sources** - Easy to add new data sources
- **Keyboard-driven** - Navigate and act without touching the mouse
- **Fast** - Single binary, instant startup

## Installation

```bash
# With Go
go install github.com/jcornudella/digest/cmd/digest@latest

# Or build from source
git clone https://github.com/jcornudella/digest
cd digest
go build -o digest ./cmd/digest
```

## Usage

```bash
# Show your digest
digest

# Initialize config
digest config --init

# List themes
digest themes

# Show help
digest help
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `↑` / `↓` | Navigate up/down |
| `tab` | Next section |
| `shift+tab` | Previous section |
| `enter` / `e` | Expand/collapse item |
| `o` | Open in browser |
| `c` | Open comments (HN) |
| `r` | Refresh |
| `1-9` | Jump to section |
| `q` / `esc` | Quit |

## Configuration

Config file location: `~/.config/digest/digest.yaml`

```yaml
theme: synthwave

sources:
  hackernews:
    enabled: true
    settings:
      max: 8
```

## Themes

### Synthwave (default)
Vibrant neon pinks, purples, and cyans. Retro-futuristic vibes.

### Nord
Cool arctic blues. Clean and minimal.

### Dracula
Dark purples and pinks. Classic dark theme.

## Adding Sources

Sources implement the `Source` interface:

```go
type Source interface {
    Name() string
    Icon() string
    Fetch(ctx context.Context, cfg Config) (*Section, error)
    TTL() time.Duration
}
```

See `internal/sources/hackernews/hackernews.go` for an example.

## Roadmap

- [ ] GitHub integration (PRs, issues, notifications)
- [ ] Linear integration
- [ ] Google Calendar integration
- [ ] Slack integration
- [ ] RSS feeds
- [ ] AI summarization
- [ ] Custom themes
- [ ] Plugin system

## License

MIT
