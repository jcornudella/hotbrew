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
	"github.com/jcornudella/hotbrew/internal/sources/hackernews"
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

	// Navigation
	sectionIdx int
	itemIdx    int
	expanded   bool

	// UI
	spinner spinner.Model
	width   int
	height  int
}

// Messages
type sectionsLoadedMsg struct {
	sections []*source.Section
}

type errorMsg struct {
	err error
}

type tickMsg time.Time

// NewModel creates a new application model
func NewModel(cfg *config.Config) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6ad5"))

	return Model{
		cfg:     cfg,
		theme:   theme.Get(cfg.Theme),
		state:   StateLoading,
		spinner: s,
		width:   80,
		height:  24,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		fetchSections(m.cfg),
	)
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

		// Add more sources here as they're implemented...

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

	case "r":
		// Refresh
		m.state = StateLoading
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

	// Header
	b.WriteString(components.Header(m.theme, m.width))
	b.WriteString("\n\n")

	switch m.state {
	case StateLoading:
		b.WriteString(m.renderLoading())

	case StateError:
		b.WriteString(m.renderError())

	case StateReady:
		b.WriteString(m.renderSections())
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
		rendered := components.Section(section, m.theme, m.width, selectedIdx, isSelected)
		parts = append(parts, rendered)
	}

	return strings.Join(parts, "")
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
