package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jcornudella/hotbrew/server"
)

func (r *Root) cmdServe(args []string) error {
	addr := ":8080"
	if len(args) > 0 {
		addr = args[0]
	}

	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".config", "hotbrew", "server")

	fmt.Println("☕ Starting hotbrew server...")
	return server.Run(addr, dataDir)
}
