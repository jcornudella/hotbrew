package cmd

import (
	"github.com/jcornudella/hotbrew/internal/cli"
	"github.com/jcornudella/hotbrew/internal/store"
)

func (r *Root) cmdMute(args []string) error {
	domain := ""
	if len(args) > 0 {
		domain = args[0]
	}
	return withStore(func(st *store.Store) error {
		cli.Mute(st, domain)
		return nil
	})
}

func (r *Root) cmdBoost(args []string) error {
	tag := ""
	if len(args) > 0 {
		tag = args[0]
	}
	return withStore(func(st *store.Store) error {
		cli.Boost(st, tag)
		return nil
	})
}

func (r *Root) cmdRules(args []string) error {
	return withStore(func(st *store.Store) error {
		if len(args) > 1 && args[0] == "--delete" {
			cli.DeleteRule(st, args[1])
			return nil
		}
		cli.Rules(st)
		return nil
	})
}

func (r *Root) cmdSources(args []string) error {
	return withStore(func(st *store.Store) error {
		cli.Sources(st)
		return nil
	})
}
