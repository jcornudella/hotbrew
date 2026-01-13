// Package source defines the interface for digest data sources.
// Implement this interface to create new sources (GitHub, Linear, Calendar, etc.)
package source

import (
	"context"
	"time"
)

// Priority levels for items
type Priority int

const (
	Low Priority = iota
	Medium
	High
	Urgent
)

func (p Priority) String() string {
	switch p {
	case Urgent:
		return "urgent"
	case High:
		return "high"
	case Medium:
		return "medium"
	default:
		return "low"
	}
}

// Action represents something the user can do with an item
type Action struct {
	Key     string // Keyboard shortcut, e.g., "o"
	Label   string // Human readable, e.g., "open in browser"
	Command string // What to execute
}

// Item represents a single piece of information from a source
type Item struct {
	ID        string
	Title     string
	Subtitle  string
	Body      string
	URL       string
	Priority  Priority
	Timestamp time.Time
	Category  string // e.g., "calendar", "pr", "news"
	Icon      string // emoji or nerd font icon
	Actions   []Action
	Metadata  map[string]any
}

// Section represents a group of items from a source
type Section struct {
	Name     string
	Icon     string
	Priority int // For ordering sections
	Items    []Item
}

// Config holds source-specific configuration
type Config struct {
	Enabled  bool
	Settings map[string]any
}

// Source is the interface all data sources must implement
type Source interface {
	// Name returns the display name of the source
	Name() string

	// Icon returns an emoji or icon for the source
	Icon() string

	// Fetch retrieves items from the source
	Fetch(ctx context.Context, cfg Config) (*Section, error)

	// TTL returns how long to cache results
	TTL() time.Duration
}

// Registry holds all available sources
type Registry struct {
	sources map[string]Source
}

// NewRegistry creates a new source registry
func NewRegistry() *Registry {
	return &Registry{
		sources: make(map[string]Source),
	}
}

// Register adds a source to the registry
func (r *Registry) Register(name string, s Source) {
	r.sources[name] = s
}

// Get retrieves a source by name
func (r *Registry) Get(name string) (Source, bool) {
	s, ok := r.sources[name]
	return s, ok
}

// All returns all registered sources
func (r *Registry) All() map[string]Source {
	return r.sources
}
