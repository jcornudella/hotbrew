package sinks

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jcornudella/hotbrew/pkg/trss"
)

// StreamLog appends digest items to a log file, one NDJSON line per item.
type StreamLog struct {
	Path string
}

func (s *StreamLog) Deliver(d *trss.Digest) error {
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	f, err := os.OpenFile(s.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open stream log: %w", err)
	}
	defer f.Close()

	return trss.EncodeItems(f, d.Items)
}
