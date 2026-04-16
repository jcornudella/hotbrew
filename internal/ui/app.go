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
	"github.com/jcornudella/hotbrew/internal/app"
	"github.com/jcornudella/hotbrew/internal/briefing"
	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/intel"
	"github.com/jcornudella/hotbrew/internal/sinks"
	"github.com/jcornudella/hotbrew/internal/store"
	"github.com/jcornudella/hotbrew/internal/ui/components"
	"github.com/jcornudella/hotbrew/internal/ui/theme"
	"github.com/jcornudella/hotbrew/pkg/profile"
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
	briefing *intel.Briefing
	err      error
	store    *store.Store

	// Navigation
	sectionIdx int
	itemIdx    int
	expanded   bool

	// Overlay for explain / why. Both reuse the same buffer since
	// only one overlay is visible at a time.
	overlayTitle string
	overlayText  string

	// UI
	spinner             spinner.Model
	width               int
	height              int
	animFrame           int
	statusMsg           string
	themePicker         bool
	themeList           []string
	themeCursor         int
	previewTheme        string
	profilePicker       bool
	profileList         []profile.Info
	profileCursor       int
	profileEditor       bool
	profileEditorName   string
	profileEditorSpecs  []profile.SourceSpec
	profileEditorState  []bool
	profileEditorCursor int
}

// Messages
type sectionsLoadedMsg struct {
	sections []*source.Section
	briefing *intel.Briefing
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
		cfg:           cfg,
		theme:         theme.Get(cfg.Theme),
		state:         StateLoading,
		spinner:       s,
		width:         80,
		height:        24,
		themeList:     theme.List(),
		profileList:   profile.List(),
		profileCursor: 0,
	}
	if len(st) > 0 && st[0] != nil {
		m.store = st[0]
	}
	return m
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	loadCmd := unavailableStoreMsg()
	if m.store != nil {
		loadCmd = loadFromStore(m.store, m.cfg, app.BuildOptions{SyncIfEmpty: true})
	}
	return tea.Batch(
		m.spinner.Tick,
		loadCmd,
		animTick(),
	)
}

// loadFromStore builds a theme-assembled briefing and flattens it
// into TUI sections. The full briefing is kept on the message so the
// UI can hand it to explain/why overlays without re-running the
// pipeline.
func loadFromStore(st *store.Store, cfg *config.Config, opts app.BuildOptions) tea.Cmd {
	return func() tea.Msg {
		service := app.NewBriefingService(st, cfg)
		b, err := service.Build(context.Background(), opts)
		if err != nil {
			return errorMsg{err: err}
		}
		if b == nil || len(b.Items) == 0 {
			return sectionsLoadedMsg{sections: nil, briefing: b}
		}
		return sectionsLoadedMsg{
			sections: sinks.BriefingToSections(b),
			briefing: b,
		}
	}
}

// animTick returns a command that ticks the animation
func animTick() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
		return animTickMsg(t)
	})
}

func unavailableStoreMsg() tea.Cmd {
	return func() tea.Msg {
		return errorMsg{err: fmt.Errorf("store unavailable")}
	}
}

// Update handles messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.profileEditor {
			return m.handleProfileEditorKey(msg)
		}
		if m.profilePicker {
			return m.handleProfileKey(msg)
		}
		if m.themePicker {
			return m.handleThemeKey(msg)
		}
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
		m.briefing = msg.briefing
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
	// An explain/why overlay takes precedence — esc (or another
	// explain key) dismisses it without affecting navigation.
	if m.overlayText != "" {
		if key := msg.String(); key == "esc" || key == "q" {
			m.overlayText = ""
			m.overlayTitle = ""
			return m, nil
		}
	}

	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit

	case "j", "down":
		m = m.moveDown()

	case "k", "up":
		m = m.moveUp()

	case "enter":
		m.expanded = !m.expanded

	case "e":
		m = m.openExplainOverlay()

	case "?":
		m = m.openWhyOverlay()

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
			return m, loadFromStore(m.store, m.cfg, app.BuildOptions{ForceSync: true})
		}
		return m, unavailableStoreMsg()

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
	case "t", "T":
		m = m.startThemePicker()
		return m, nil
	case "p", "P":
		m = m.startProfilePicker()
		return m, nil
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

func (m Model) startThemePicker() Model {
	m.themePicker = true
	m.profilePicker = false
	m.profileEditor = false
	m.previewTheme = m.cfg.Theme
	for i, name := range m.themeList {
		if name == m.cfg.Theme {
			m.themeCursor = i
			break
		}
	}
	m = m.applyPreviewTheme(m.themeList[m.themeCursor])
	return m
}

func (m Model) startProfilePicker() Model {
	m.profilePicker = true
	m.themePicker = false
	m.profileEditor = false
	m.profileList = profile.List()
	if len(m.profileList) == 0 {
		m.profileList = []profile.Info{{Name: m.cfg.GetProfileName(), SourceCount: 0}}
	}
	selected := m.cfg.GetProfileName()
	m.profileCursor = 0
	for i, info := range m.profileList {
		if info.Name == selected {
			m.profileCursor = i
			break
		}
	}
	return m
}

func (m Model) startProfileEditor(name string) Model {
	if name == "" {
		return m
	}
	p := profile.Load(name)
	if p == nil {
		return m
	}
	m.profileEditor = true
	m.profilePicker = false
	m.profileEditorName = name
	m.profileEditorCursor = 0
	m.profileEditorSpecs = make([]profile.SourceSpec, len(p.Sources))
	copy(m.profileEditorSpecs, p.Sources)
	m.profileEditorState = make([]bool, len(m.profileEditorSpecs))
	for i := range m.profileEditorState {
		m.profileEditorState[i] = true
	}
	return m
}

func (m Model) handleThemeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "h":
		if len(m.themeList) > 0 {
			m.themeCursor = (m.themeCursor - 1 + len(m.themeList)) % len(m.themeList)
			m = m.applyPreviewTheme(m.themeList[m.themeCursor])
		}
	case "right", "l":
		if len(m.themeList) > 0 {
			m.themeCursor = (m.themeCursor + 1) % len(m.themeList)
			m = m.applyPreviewTheme(m.themeList[m.themeCursor])
		}
	case "enter":
		if len(m.themeList) > 0 {
			selected := m.themeList[m.themeCursor]
			m.themePicker = false
			return m.applyTheme(selected)
		}
	case "esc":
		m.themePicker = false
		m = m.applyPreviewTheme(m.cfg.Theme)
	}
	return m, nil
}

func (m Model) handleProfileKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "h", "up", "k":
		if len(m.profileList) > 0 {
			m.profileCursor = (m.profileCursor - 1 + len(m.profileList)) % len(m.profileList)
		}
	case "right", "l", "down", "j":
		if len(m.profileList) > 0 {
			m.profileCursor = (m.profileCursor + 1) % len(m.profileList)
		}
	case "enter":
		if len(m.profileList) > 0 {
			selected := m.profileList[m.profileCursor].Name
			return m.applyProfile(selected)
		}
	case "e":
		if len(m.profileList) > 0 {
			return m.startProfileEditor(m.profileList[m.profileCursor].Name), nil
		}
	case "esc":
		m.profilePicker = false
	}
	return m, nil
}

func (m Model) handleProfileEditorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if l := len(m.profileEditorSpecs); l > 0 {
			m.profileEditorCursor = (m.profileEditorCursor - 1 + l) % l
		}
	case "down", "j":
		if l := len(m.profileEditorSpecs); l > 0 {
			m.profileEditorCursor = (m.profileEditorCursor + 1) % l
		}
	case "space", "enter":
		if len(m.profileEditorState) > 0 && m.profileEditorCursor < len(m.profileEditorState) {
			m.profileEditorState[m.profileEditorCursor] = !m.profileEditorState[m.profileEditorCursor]
		}
	case "s":
		return m.saveProfileEditor()
	case "esc":
		m.profileEditor = false
		return m, nil
	}
	return m, nil
}

func (m Model) applyPreviewTheme(name string) Model {
	if name == "" {
		return m
	}
	m.theme = theme.Get(name)
	m.previewTheme = name
	return m
}

func (m Model) applyTheme(name string) (tea.Model, tea.Cmd) {
	m.cfg.Theme = name
	m.theme = theme.Get(name)
	m.previewTheme = ""
	m.themePicker = false
	config.Save(m.cfg)
	m.statusMsg = fmt.Sprintf("Theme switched to %s", name)
	return m, animTick()
}

func (m Model) applyProfile(name string) (tea.Model, tea.Cmd) {
	if name == "" {
		m.profilePicker = false
		return m, nil
	}
	m.profilePicker = false
	if m.cfg.Profile == name {
		return m, nil
	}
	m.cfg.Profile = name
	config.Save(m.cfg)
	m.statusMsg = fmt.Sprintf("Profile switched to %s", name)
	m.state = StateLoading
	if m.store != nil {
		return m, loadFromStore(m.store, m.cfg, app.BuildOptions{SyncIfEmpty: true, ForceSync: true})
	}
	return m, unavailableStoreMsg()
}

func (m Model) saveProfileEditor() (tea.Model, tea.Cmd) {
	var updated []profile.SourceSpec
	for i, spec := range m.profileEditorSpecs {
		if i < len(m.profileEditorState) && m.profileEditorState[i] {
			updated = append(updated, spec)
		}
	}
	if err := profile.Save(m.profileEditorName, updated); err != nil {
		m.statusMsg = fmt.Sprintf("Save failed: %v", err)
		return m, nil
	}
	m.profileEditor = false
	m.profileList = profile.List()
	return m.applyProfile(m.profileEditorName)
}

// openExplainOverlay builds the user-facing explanation for the
// currently selected cluster lead. If the briefing isn't available
// (first load, sync failure) we fall back to a status blurb rather
// than silently swallowing the key.
func (m Model) openExplainOverlay() Model {
	exp, ok := m.explanationForSelection()
	if !ok {
		m.statusMsg = "Explain unavailable — no briefing loaded"
		return m
	}
	m.overlayTitle = "Why it matters"
	m.overlayText = exp.WhyItMatters
	return m
}

func (m Model) openWhyOverlay() Model {
	exp, ok := m.explanationForSelection()
	if !ok {
		m.statusMsg = "Why unavailable — no briefing loaded"
		return m
	}
	var lines []string
	lines = append(lines, exp.WhyYouSee)
	for _, f := range exp.Factors {
		lines = append(lines, fmt.Sprintf("• %-14s x%.2f — %s", f.Name, f.Multiplier, f.Description))
	}
	m.overlayTitle = "Why you're seeing this"
	m.overlayText = strings.Join(lines, "\n")
	return m
}

func (m Model) explanationForSelection() (briefing.Explanation, bool) {
	if m.briefing == nil {
		return briefing.Explanation{}, false
	}
	item := m.selectedItem()
	if item == nil {
		return briefing.Explanation{}, false
	}
	id := item.ID
	if trssID, ok := item.Metadata["trss_id"].(string); ok && trssID != "" {
		id = trssID
	}
	return briefing.Explain(m.briefing, id)
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
		content := m.renderSections()
		overlay := ""
		switch {
		case m.profileEditor:
			overlay = m.renderProfileEditor()
		case m.profilePicker:
			overlay = m.renderProfilePicker()
		case m.themePicker:
			overlay = m.renderThemePicker()
		case m.overlayText != "":
			overlay = m.renderExplainOverlay()
		}
		if overlay != "" {
			content = m.applyOverlay(content, overlay)
		}
		b.WriteString(content)
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
	current, total := m.progressCounts()
	b.WriteString("\n")
	b.WriteString(components.Footer(m.theme, m.width, m.expanded, current, total))

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

func (m Model) progressCounts() (int, int) {
	total := 0
	current := 0
	for si, section := range m.sections {
		if section == nil {
			continue
		}
		count := len(section.Items)
		if count == 0 {
			continue
		}
		total += count
		if si < m.sectionIdx {
			current += count
			continue
		}
		if si == m.sectionIdx {
			idx := m.itemIdx
			if idx >= count {
				idx = count - 1
			}
			current += idx + 1
		}
	}
	if total == 0 {
		return 0, 0
	}
	if current < 1 {
		current = 1
	}
	if current > total {
		current = total
	}
	return current, total
}

func (m Model) applyOverlay(content, overlay string) string {
	if overlay == "" {
		return content
	}
	height := lipgloss.Height(overlay)
	if height <= 0 {
		return content
	}
	placed := lipgloss.Place(
		m.width,
		height,
		lipgloss.Center,
		lipgloss.Bottom,
		overlay,
		lipgloss.WithWhitespaceChars(" "),
	)
	base := strings.TrimRight(content, "\n")
	if base != "" {
		base += "\n\n"
	}
	return base + placed
}

func (m Model) renderThemePicker() string {
	if len(m.themeList) == 0 {
		return ""
	}

	header := m.theme.AccentStyle().Bold(true).Render("Theme Picker")
	instructions := m.theme.MutedStyle().Render("←/→ preview  •  enter apply  •  esc cancel")
	var options []string
	for i, name := range m.themeList {
		optionTheme := theme.Get(name)
		swatch := renderThemeSwatch(optionTheme.HeaderGradient())
		label := fmt.Sprintf("%s %s", swatch, name)
		style := m.theme.MutedStyle()
		if name == m.cfg.Theme {
			style = style.Bold(true)
		}
		if i == m.themeCursor {
			style = m.theme.ItemSelectedStyle().Bold(true)
		}
		options = append(options, style.Render(label))
	}

	body := strings.Join(options, "\n")
	width := m.width / 2
	if width < 32 {
		width = m.width - 4
	}
	if width < 20 {
		width = m.width
	}
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Accent()).
		Padding(1, 3).
		Width(width)

	return box.Render(lipgloss.JoinVertical(lipgloss.Left, header, instructions, "", body))
}

func (m Model) renderProfilePicker() string {
	if len(m.profileList) == 0 {
		return ""
	}

	header := m.theme.AccentStyle().Bold(true).Render("Profile Picker")
	instructions := m.theme.MutedStyle().Render("↑/↓ select  •  enter apply  •  e edit  •  esc cancel")
	current := m.cfg.GetProfileName()
	var rows []string
	for i, info := range m.profileList {
		label := info.Name
		if info.SourceCount > 0 {
			label = fmt.Sprintf("%s  •  %d sources", info.Name, info.SourceCount)
		}
		style := m.theme.MutedStyle()
		if info.Name == current {
			style = style.Bold(true)
		}
		if i == m.profileCursor {
			style = m.theme.ItemSelectedStyle().Bold(true)
		}
		rows = append(rows, style.Render(label))
	}

	body := strings.Join(rows, "\n")
	width := m.width / 2
	if width < 32 {
		width = m.width - 4
	}
	if width < 24 {
		width = m.width
	}
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Accent()).
		Padding(1, 3).
		Width(width)

	return box.Render(lipgloss.JoinVertical(lipgloss.Left, header, instructions, "", body))
}

func (m Model) renderProfileEditor() string {
	if !m.profileEditor {
		return ""
	}

	header := m.theme.AccentStyle().Bold(true).Render(fmt.Sprintf("Editing: %s", m.profileEditorName))
	instructions := m.theme.MutedStyle().Render("space toggle  •  s save  •  esc cancel")
	if len(m.profileEditorSpecs) == 0 {
		placeholder := m.theme.MutedStyle().Render("Profile is empty. Save to keep it blank or esc to cancel.")
		box := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(m.theme.Accent()).
			Padding(1, 3).
			Width(m.width / 2)
		return box.Render(lipgloss.JoinVertical(lipgloss.Left, header, instructions, "", placeholder))
	}

	var rows []string
	for i, spec := range m.profileEditorSpecs {
		checked := "[ ]"
		if i < len(m.profileEditorState) && m.profileEditorState[i] {
			checked = "[x]"
		}
		label := fmt.Sprintf("%s %s (%s)", checked, spec.Name, spec.Driver)
		style := m.theme.MutedStyle()
		if i == m.profileEditorCursor {
			style = m.theme.ItemSelectedStyle().Bold(true)
		}
		rows = append(rows, style.Render(label))
	}

	body := strings.Join(rows, "\n")
	width := m.width / 2
	if width < 40 {
		width = m.width - 4
	}
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Accent()).
		Padding(1, 3).
		Width(width)

	return box.Render(lipgloss.JoinVertical(lipgloss.Left, header, instructions, "", body))
}

// renderExplainOverlay draws a bordered box with the current
// overlay title and body. Both explain ("why it matters") and why
// ("why you're seeing this") share this renderer — they differ only
// in what text is stashed in overlayText.
func (m Model) renderExplainOverlay() string {
	if m.overlayText == "" {
		return ""
	}
	header := m.theme.AccentStyle().Bold(true).Render(m.overlayTitle)
	instructions := m.theme.MutedStyle().Render("esc close")
	body := m.theme.ItemStyle().Render(m.overlayText)

	width := m.width * 2 / 3
	if width < 40 {
		width = m.width - 4
	}
	if width < 24 {
		width = m.width
	}
	box := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Accent()).
		Padding(1, 3).
		Width(width)

	return box.Render(lipgloss.JoinVertical(lipgloss.Left, header, instructions, "", body))
}

func renderThemeSwatch(colors []string) string {
	if len(colors) == 0 {
		return ""
	}
	max := 4
	if len(colors) < max {
		max = len(colors)
	}
	var blocks []string
	for i := 0; i < max; i++ {
		block := lipgloss.NewStyle().Foreground(lipgloss.Color(colors[i])).Render("██")
		blocks = append(blocks, block)
	}
	return strings.Join(blocks, "")
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
