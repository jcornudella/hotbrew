# ☕ hotbrew

**Your morning, piping hot.**

A beautiful terminal newsletter that aggregates your daily information into a single, scannable digest.

![CI](https://github.com/jcornudella/hotbrew/workflows/CI/badge.svg)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- **Beautiful UI** - Gradient text, themed colors, smooth interactions
- **Multiple themes** - Synthwave (neon), Nord (arctic), Dracula (dark)
- **Pluggable sources** - Easy to add new data sources
- **Keyboard-driven** - Navigate and act without the mouse
- **Fast** - Single binary, instant startup

## Installation

```bash
# With Go
go install github.com/jcornudella/hotbrew/cmd/hotbrew@latest

# Build from source
git clone https://github.com/jcornudella/hotbrew.git
cd hotbrew
make build
./hotbrew
```

## Usage

```bash
# Show your digest
hotbrew

# Initialize config
hotbrew config --init

# List themes
hotbrew themes

# Show help
hotbrew help
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

Config file: `~/.config/hotbrew/hotbrew.yaml`

```yaml
# Theme: synthwave, nord, dracula
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

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## Roadmap

- [ ] GitHub integration (PRs, issues, notifications)
- [ ] Linear integration
- [ ] Google Calendar
- [ ] Slack highlights
- [ ] RSS feeds
- [ ] AI summarization
- [ ] Custom themes
- [ ] Plugin system

## Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT - see [LICENSE](LICENSE)

---

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).
