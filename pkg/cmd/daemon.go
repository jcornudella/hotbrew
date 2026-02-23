package cmd

import (
	"fmt"

	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/daemon"
)

func (r *Root) cmdDaemon(args []string) error {
	action := "status"
	if len(args) > 0 {
		action = args[0]
	}

	switch action {
	case "start":
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}
		registry := buildRegistry(cfg)
		if err := daemon.Start(cfg, registry); err != nil {
			return err
		}
	case "stop":
		if err := daemon.Stop(); err != nil {
			return err
		}
	case "status":
		daemon.Status()
	default:
		fmt.Println("Usage: hotbrew daemon [start|stop|status]")
	}
	return nil
}
