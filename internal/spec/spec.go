// Package spec defines the StreamSpec interface and concrete specifications
// for different MikroTik API streaming commands. Each spec file encapsulates
// the command construction, response parsing, and domain mapping for a specific
// data source, following the Command pattern.
package spec

import (
	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
)

// StreamSpec defines the contract for a MikroTik API streaming command specification.
// Each implementation represents a single data source (e.g., interface stats, active users)
// and knows how to:
//   - Construct the API command with optimal proplist
//   - Parse raw API responses into domain events
//   - Provide metadata for InfluxDB storage
//
// This interface enables the collector to manage multiple data streams uniformly
// without knowing the details of each command.
type StreamSpec interface {
	// Command returns the RouterOS API command arguments as a string slice.
	// Must include .proplist to limit data sent by the router for CPU efficiency.
	// Example: []string{"/interface/print", "=stats", "=interval=1s", "=.proplist=name,type,rx-byte,tx-byte"}
	Command() []string

	// Tag returns a unique identifier for this streaming command.
	// Used internally by the go-routeros library for response demultiplexing
	// when multiple commands run on the same async connection.
	Tag() string

	// Measurement returns the InfluxDB measurement name for data from this spec.
	// Example: "interface_stats", "hotspot_active"
	Measurement() string

	// Parse converts a raw RouterOS API sentence into a TelemetryEvent.
	// routerID identifies which router produced this sentence.
	// Returns nil if the sentence should be skipped (e.g., empty or irrelevant).
	// Returns an EventDead type if the sentence contains .dead=yes.
	Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error)
}
