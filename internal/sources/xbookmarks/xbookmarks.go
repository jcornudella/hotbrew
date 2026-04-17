// Package xbookmarks provides a source connector for X (Twitter) bookmarks.
//
// Scaffolded shell: the Fetch method returns a single placeholder item
// instructing the user to run `hotbrew x-auth`. Real API calls land in
// a follow-up PR once the PKCE flow and token persistence are wired up.
package xbookmarks

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/jcornudella/hotbrew/pkg/source"
)

const (
	tokenFilename = "x_token.json"
	defaultMax    = 20
)

// Source fetches a user's X bookmarks.
type Source struct {
	name string
	icon string
}

// New creates an X bookmarks source.
func New(name, icon string) *Source {
	if name == "" {
		name = "X Bookmarks"
	}
	if icon == "" {
		icon = "🔖"
	}
	return &Source{name: name, icon: icon}
}

// Name implements source.Source.
func (s *Source) Name() string { return s.name }

// Icon implements source.Source.
func (s *Source) Icon() string { return s.icon }

// TTL implements source.Source.
func (s *Source) TTL() time.Duration { return 30 * time.Minute }

// Fetch returns the user's recent bookmarks, or a single "needs auth"
// item when no token is on disk yet.
func (s *Source) Fetch(ctx context.Context, cfg source.Config) (*source.Section, error) {
	if !tokenExists() {
		return s.needsAuthSection(), nil
	}
	// Real fetch lands in PR C.
	return &source.Section{
		Name:  s.name,
		Icon:  s.icon,
		Items: nil,
	}, nil
}

func (s *Source) needsAuthSection() *source.Section {
	return &source.Section{
		Name: s.name,
		Icon: s.icon,
		Items: []source.Item{
			{
				ID:       "xbookmarks-needs-auth",
				Title:    "Connect your X account",
				Subtitle: "Run `hotbrew x-auth` to authorize hotbrew to read your bookmarks",
				Priority: source.Medium,
				Category: "auth",
				Icon:     s.icon,
			},
		},
	}
}

// TokenPath returns the on-disk location of the stored OAuth token.
func TokenPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "hotbrew", tokenFilename)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "hotbrew", tokenFilename)
}

func tokenExists() bool {
	_, err := os.Stat(TokenPath())
	return err == nil
}
