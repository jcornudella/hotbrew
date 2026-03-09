package app

import (
	"context"

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

func syncStore(ctx context.Context, st *store.Store, cfg *config.Config) error {
	if st == nil || cfg == nil {
		return nil
	}
	registry := buildRegistry(cfg)
	hsync.SyncAll(ctx, st, registry)
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
