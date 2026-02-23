package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/sources/arxiv"
	"github.com/jcornudella/hotbrew/internal/sources/github"
	"github.com/jcornudella/hotbrew/internal/sources/hackernews"
	"github.com/jcornudella/hotbrew/internal/sources/hnsearch"
	"github.com/jcornudella/hotbrew/internal/sources/lobsters"
	"github.com/jcornudella/hotbrew/internal/sources/reddit"
	"github.com/jcornudella/hotbrew/internal/sources/tldr"
	"github.com/jcornudella/hotbrew/internal/store"
	hsync "github.com/jcornudella/hotbrew/internal/sync"
	"github.com/jcornudella/hotbrew/pkg/profile"
	"github.com/jcornudella/hotbrew/pkg/source"
)

func (r *Root) cmdSync(args []string) error {
	if len(args) > 0 && args[0] == "--remote" {
		return r.cmdSyncRemote()
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	st, err := store.Open(cfg.GetDBPath())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	registry := buildRegistry(cfg)

	fmt.Println("☕ Syncing sources...")
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	results := hsync.SyncAll(ctx, st, registry)
	hsync.PrintResults(results)

	fmt.Printf("\nTotal items in store: %d\n", st.ItemCount())
	return nil
}

func (r *Root) cmdSyncRemote() error {
	home, _ := os.UserHomeDir()
	tokenPath := filepath.Join(home, ".config", "hotbrew", "token")

	tokenData, err := os.ReadFile(tokenPath)
	if err != nil {
		fmt.Println("Not logged in. Run 'hotbrew login <token>' first.")
		fmt.Println("Get your token at https://hotbrew.dev")
		return nil
	}

	token := strings.TrimSpace(string(tokenData))

	serverURL := os.Getenv("HOTBREW_SERVER")
	if serverURL == "" {
		serverURL = "https://hotbrew.dev"
	}

	resp, err := http.Get(serverURL + "/api/config/" + token)
	if err != nil {
		return fmt.Errorf("connect server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		fmt.Println("Invalid token. Please check your token or subscribe at https://hotbrew.dev")
		return nil
	}

	var remoteCfg map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&remoteCfg); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	fmt.Println("☕ Remote config synced!")
	fmt.Printf("Theme: %v\n", remoteCfg["theme"])
	return nil
}

func buildRegistry(cfg *config.Config) *source.Registry {
	registry := source.NewRegistry()
	prof := profile.Load(cfg.GetProfileName())
	for _, spec := range prof.Sources {
		if spec.ConfigKey != "" {
			if srcCfg, ok := cfg.Sources[spec.ConfigKey]; !ok || !srcCfg.Enabled {
				continue
			}
		}

		src := instantiateSource(spec)
		if src == nil {
			continue
		}
		registry.Register(spec.Key, src)
	}
	return registry
}

func instantiateSource(spec profile.SourceSpec) source.Source {
	switch spec.Driver {
	case "hackernews":
		return hackernews.New()
	case "hnsearch":
		return hnsearch.New(spec.Name, spec.Queries, spec.Icon)
	case "github-trending":
		return github.New(spec.Name, spec.Topics, spec.Icon)
	case "tldr":
		if spec.FeedURL == "" {
			return nil
		}
		return tldr.New(spec.Name, spec.FeedURL, spec.Icon)
	case "lobsters":
		return lobsters.New(spec.Name, spec.Tags, spec.Icon)
	case "reddit":
		return reddit.New(spec.Name, spec.Subreddits, spec.Icon)
	case "arxiv":
		cats := spec.Categories
		if len(cats) == 0 {
			cats = arxiv.DefaultCategories
		}
		return arxiv.New(spec.Name, cats, spec.Icon)
	default:
		return nil
	}
}
