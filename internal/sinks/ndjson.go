package sinks

import (
	"io"
	"os"

	"github.com/jcornudella/hotbrew/pkg/trss"
)

// NDJSON writes the digest as NDJSON to a writer.
type NDJSON struct {
	Writer io.Writer
}

// NewNDJSON creates an NDJSON sink writing to stdout.
func NewNDJSON() *NDJSON {
	return &NDJSON{Writer: os.Stdout}
}

func (n *NDJSON) Deliver(d *trss.Digest) error {
	return trss.EncodeDigest(n.Writer, d)
}
