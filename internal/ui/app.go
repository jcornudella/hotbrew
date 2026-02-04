// Package ui contains the main Bubble Tea application
package ui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/curation"
	"github.com/jcornudella/hotbrew/internal/sinks"
	"github.com/jcornudella/hotbrew/internal/sources/github"
	"github.com/jcornudella/hotbrew/internal/sources/hackernews"
	"github.com/jcornudella/hotbrew/internal/sources/hnsearch"
	"github.com/jcornudella/hotbrew/internal/store"
	"github.com/jcornudella/hotbrew/internal/ui/components"
	"github.com/jcornudella/hotbrew/internal/ui/theme"
	"github.com/jcornudella/hotbrew/pkg/source"
)

// State represents the app state
type State int

const (
	StateLoading State = iota
	StateReady
	StateError
)

// Model is the main application model
type Model struct {
	cfg      *config.Config
	theme    theme.Theme
	state    State
	sections []*source.Section
	err      error
	store    *store.Store

	// Navigation
	sectionIdx int
	itemIdx    int
	expanded   bool

	// UI
	spinner    spinner.Model
	width      int
	height     int
	animFrame  int
	statusMsg  string
}

// Messages
type sectionsLoadedMsg struct {
	sections []*source.Section
}

type errorMsg struct {
	err error
}

type tickMsg time.Time
type animTickMsg time.Time

// NewModel creates a new application model.
// If st is non-nil, the TUI loads from the curation engine.
func NewModel(cfg *config.Config, st ...*store.Store) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6ad5"))

	// Register custom theme if configured
	if cfg.Theme == "custom" && cfg.CustomTheme != nil {
		theme.RegisterCustom("custom", theme.CustomColors{
			Primary:        cfg.CustomTheme.Primary,
			Secondary:      cfg.CustomTheme.Secondary,
			Accent:         cfg.CustomTheme.Accent,
			Muted:          cfg.CustomTheme.Muted,
			Background:     cfg.CustomTheme.Background,
			Text:           cfg.CustomTheme.Text,
			TextMuted:      cfg.CustomTheme.TextMuted,
			HeaderGradient: cfg.CustomTheme.HeaderGradient,
		})
	}

	m := Model{
		cfg:     cfg,
		theme:   theme.Get(cfg.Theme),
		state:   StateLoading,
		spinner: s,
		width:   80,
		height:  24,
	}
	if len(st) > 0 && st[0] != nil {
		m.store = st[0]
	}
	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	loadCmd := fetchSections(m.cfg)
	if m.store != nil {
		loadCmd = loadFromStore(m.store, m.cfg)
	}
	return tea.Batch(
		m.spinner.Tick,
		loadCmd,
		animTick(),
	)
}

// loadFromStore generates a digest from the store and converts to sections.
func loadFromStore(st *store.Store, cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		engine := curation.NewEngine(st)
		digest, err := engine.GenerateDigest(cfg.GetDigestWindow(), cfg.GetDigestMax(), "Hotbrew Digest")
		if err != nil || digest == nil || len(digest.Items) == 0 {
			// Fall back to live fetch if store is empty.
			return fetchSections(cfg)()
		}

		sections := sinks.DigestToSections(digest)
		if len(sections) == 0 {
			return fetchSections(cfg)()
		}

		return sectionsLoadedMsg{sections: sections}
	}
}

// animTick returns a command that ticks the animation
func animTick() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return animTickMsg(t)
	})
}

// fetchSections fetches data from all enabled sources
func fetchSections(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var sections []*source.Section

		// Fetch Hacker News if enabled
		if src, ok := cfg.Sources["hackernews"]; ok && src.Enabled {
			hn := hackernews.New()
			section, err := hn.Fetch(ctx, source.Config{
				Enabled:  src.Enabled,
				Settings: src.Settings,
			})
			if err == nil && section != nil {
				sections = append(sections, section)
			}
		}

		// Claude Code & Vibe Coding search
		vibeSearch := hnsearch.New(
			"Claude Code & Vibe Coding",
			[]string{"Claude Code", "vibe coding", "AI coding assistant", "Anthropic Claude"},
			"🤖",
		)
		vibeSection, vibeErr := vibeSearch.Fetch(ctx, source.Config{
			Enabled: true,
			Settings: map[string]any{"max": 8},
		})
		if vibeErr == nil && vibeSection != nil && len(vibeSection.Items) > 0 {
			sections = append(sections, vibeSection)
		}

		// GitHub trending AI/coding repos
		ghTrending := github.New(
			"GitHub Trending",
			[]string{"ai", "llm", "machine-learning", "gpt", "claude"},
			"🐙",
		)
		ghSection, ghErr := ghTrending.Fetch(ctx, source.Config{
			Enabled:  true,
			Settings: map[string]any{"max": 6},
		})
		if ghErr == nil && ghSection != nil && len(ghSection.Items) > 0 {
			sections = append(sections, ghSection)
		}

		return sectionsLoadedMsg{sections: sections}
	}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case sectionsLoadedMsg:
		m.sections = msg.sections
		m.state = StateReady
		return m, nil

	case errorMsg:
		m.err = msg.err
		m.state = StateError
		return m, nil

	case animTickMsg:
		m.animFrame++
		return m, animTick()
	}

	return m, nil
}

// handleKey processes keyboard input
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit

	case "j", "down":
		m = m.moveDown()

	case "k", "up":
		m = m.moveUp()

	case "enter", "e":
		m.expanded = !m.expanded

	case "o":
		// Open URL in browser
		if item := m.selectedItem(); item != nil && item.URL != "" {
			openURL(item.URL)
		}

	case "c":
		// Open comments (for HN)
		if item := m.selectedItem(); item != nil {
			if hnURL, ok := item.Metadata["hn_url"].(string); ok {
				openURL(hnURL)
			}
		}

	case "s":
		// Save item
		if m.store != nil {
			if item := m.selectedItem(); item != nil {
				id := item.ID
				if meta, ok := item.Metadata["trss_id"].(string); ok {
					id = meta
				}
				if err := m.store.MarkSaved(id); err == nil {
					m.statusMsg = "★ Saved"
				}
			}
		}

	case "u":
		// Toggle read/unread
		if m.store != nil {
			if item := m.selectedItem(); item != nil {
				id := item.ID
				if meta, ok := item.Metadata["trss_id"].(string); ok {
					id = meta
				}
				state := m.store.GetState(id)
				if state == "unread" {
					m.store.MarkRead(id)
					m.statusMsg = "✓ Marked read"
				} else {
					m.store.MarkUnread(id)
					m.statusMsg = "○ Marked unread"
				}
			}
		}

	case "m":
		// Mute domain
		if m.store != nil {
			if item := m.selectedItem(); item != nil && item.URL != "" {
				domain := extractItemDomain(item.URL)
				if domain != "" {
					m.store.AddRule("mute_domain", domain, "")
					m.statusMsg = fmt.Sprintf("🔇 Muted %s", domain)
				}
			}
		}

	case "r":
		// Refresh
		m.state = StateLoading
		m.statusMsg = ""
		if m.store != nil {
			return m, loadFromStore(m.store, m.cfg)
		}
		return m, fetchSections(m.cfg)

	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		// Quick jump to section
		idx := int(msg.String()[0] - '1')
		if idx < len(m.sections) {
			m.sectionIdx = idx
			m.itemIdx = 0
		}

	case "tab":
		// Next section
		if len(m.sections) > 0 {
			m.sectionIdx = (m.sectionIdx + 1) % len(m.sections)
			m.itemIdx = 0
		}

	case "shift+tab":
		// Previous section
		if len(m.sections) > 0 {
			m.sectionIdx = (m.sectionIdx - 1 + len(m.sections)) % len(m.sections)
			m.itemIdx = 0
		}
	}

	return m, nil
}

func (m Model) moveDown() Model {
	if len(m.sections) == 0 {
		return m
	}

	section := m.sections[m.sectionIdx]
	if m.itemIdx < len(section.Items)-1 {
		m.itemIdx++
	} else if m.sectionIdx < len(m.sections)-1 {
		// Move to next section
		m.sectionIdx++
		m.itemIdx = 0
	}
	return m
}

func (m Model) moveUp() Model {
	if len(m.sections) == 0 {
		return m
	}

	if m.itemIdx > 0 {
		m.itemIdx--
	} else if m.sectionIdx > 0 {
		// Move to previous section
		m.sectionIdx--
		m.itemIdx = len(m.sections[m.sectionIdx].Items) - 1
	}
	return m
}

func (m Model) selectedItem() *source.Item {
	if len(m.sections) == 0 || m.sectionIdx >= len(m.sections) {
		return nil
	}
	section := m.sections[m.sectionIdx]
	if m.itemIdx >= len(section.Items) {
		return nil
	}
	return &section.Items[m.itemIdx]
}

// View renders the UI
func (m Model) View() string {
	var b strings.Builder

	// Animated Header
	b.WriteString(components.AnimatedHeader(m.theme, m.width, m.animFrame))
	b.WriteString("\n\n")

	switch m.state {
	case StateLoading:
		b.WriteString(m.renderLoading())

	case StateError:
		b.WriteString(m.renderError())

	case StateReady:
		b.WriteString(m.renderSections())
	}

	// Status message
	if m.statusMsg != "" {
		statusStyle := lipgloss.NewStyle().
			Foreground(m.theme.Accent()).
			Padding(0, 2)
		b.WriteString(statusStyle.Render(m.statusMsg))
		b.WriteString("\n")
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(components.Footer(m.theme, m.width))

	return b.String()
}

func (m Model) renderLoading() string {
	style := lipgloss.NewStyle().
		Foreground(m.theme.Accent()).
		Padding(2, 0)

	return style.Render(fmt.Sprintf("%s Fetching your digest...", m.spinner.View()))
}

func (m Model) renderError() string {
	style := lipgloss.NewStyle().
		Foreground(m.theme.PriorityUrgent()).
		Padding(2, 0)

	return style.Render(fmt.Sprintf("Error: %v", m.err))
}

func (m Model) renderSections() string {
	if len(m.sections) == 0 {
		return m.theme.MutedStyle().Render("No items to display")
	}

	var parts []string
	for i, section := range m.sections {
		isSelected := i == m.sectionIdx
		selectedIdx := -1
		if isSelected {
			selectedIdx = m.itemIdx
		}

		// Render section with proper selection state
		rendered := components.Section(section, m.theme, m.width, selectedIdx, isSelected, m.expanded)
		parts = append(parts, rendered)
	}

	return strings.Join(parts, "")
}

// extractItemDomain gets the hostname from a URL.
func extractItemDomain(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	// Simple extraction without importing net/url to avoid bloat.
	// Find the host part between :// and the next /
	idx := strings.Index(rawURL, "://")
	if idx < 0 {
		return ""
	}
	host := rawURL[idx+3:]
	if slash := strings.Index(host, "/"); slash >= 0 {
		host = host[:slash]
	}
	host = strings.TrimPrefix(host, "www.")
	return host
}

// openURL opens a URL in the default browser
func openURL(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		return
	}

	_ = cmd.Start()
}
