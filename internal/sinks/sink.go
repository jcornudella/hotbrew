// Package sinks defines output targets for TRSS digests.
package sinks

import "github.com/jcornudella/hotbrew/pkg/trss"

// Sink delivers a digest to an output target.
type Sink interface {
	Deliver(digest *trss.Digest) error
}
