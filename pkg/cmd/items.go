package cmd

import (
	"fmt"
	"strings"

	"github.com/jcornudella/hotbrew/internal/cli"
	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/store"
)

func (r *Root) cmdAdd(args []string) error {
	return withStore(func(st *store.Store) error {
		cli.Add(st, args)
		return nil
	})
}

func (r *Root) cmdList(args []string) error {
	return withStore(func(st *store.Store) error { return runList(st, args) })
}

func runList(st *store.Store, args []string) error {
	opts := cli.ListOptions{Top: 20}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--unread":
			opts.Unread = true
		case "--source":
			if i+1 < len(args) {
				i++
				opts.SourceName = args[i]
			}
		case "--top":
			if i+1 < len(args) {
				i++
				fmt.Sscanf(args[i], "%d", &opts.Top)
			}
		}
	}
	cli.List(st, opts)
	return nil
}

func (r *Root) cmdOpen(args []string) error {
	var id string
	if len(args) > 0 {
		id = args[0]
	}
	return withStore(func(st *store.Store) error {
		cli.Open(st, id)
		return nil
	})
}

func (r *Root) cmdSave(args []string) error {
	var id string
	if len(args) > 0 {
		id = args[0]
	}
	return withStore(func(st *store.Store) error {
		cli.Save(st, id)
		return nil
	})
}

func (r *Root) cmdCurate(args []string) error {
	return withStore(func(st *store.Store) error { return runCurate(st, args) })
}

func runCurate(st *store.Store, args []string) error {
	opts := cli.CurateOptions{}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--title":
			if i+1 < len(args) {
				i++
				opts.Title = args[i]
			}
		case "--tags":
			if i+1 < len(args) {
				i++
				opts.Tags = strings.Split(args[i], ",")
			}
		case "--note":
			if i+1 < len(args) {
				i++
				opts.Note = args[i]
			}
		default:
			if opts.URL == "" && !strings.HasPrefix(args[i], "-") {
				opts.URL = args[i]
			}
		}
	}
	cli.Curate(st, opts)
	return nil
}

func (r *Root) cmdStream(args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	cli.Stream(cfg)
	return nil
}
