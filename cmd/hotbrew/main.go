package main

import (
	"fmt"
	"os"

	"github.com/jcornudella/hotbrew/pkg/cmd"
)

var Version = "0.1.0"

func main() {
	root := cmd.New(Version)
	args := os.Args[1:]
	if len(args) == 0 {
		args = []string{"sync-and-run"}
	}
	if err := root.Execute(args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
