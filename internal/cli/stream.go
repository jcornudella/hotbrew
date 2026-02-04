package cli

import (
	"bufio"
	"fmt"
	"os"

	"github.com/jcornudella/hotbrew/internal/config"
)

// Stream handles `hotbrew stream` — tails the stream log.
func Stream(cfg *config.Config) {
	logPath := cfg.GetStreamLogPath()

	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No stream log yet. Run 'hotbrew sync' first, or start the daemon.")
			return
		}
		fmt.Fprintf(os.Stderr, "Error opening stream log: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stream log: %v\n", err)
	}
}
