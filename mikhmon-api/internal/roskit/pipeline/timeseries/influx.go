// Package timeseries defines the InfluxDB write contract.
package timeseries

import (
	"context"
	"time"
)

// Point represents a single InfluxDB data point.
type Point struct {
	Measurement string
	Tags        map[string]string
	Fields      map[string]interface{}
	Timestamp   time.Time
}

// Writer writes time-series metrics.
// Implementations should be async and non-blocking (batch internally).
type Writer interface {
	WritePoint(ctx context.Context, point Point) error
	Flush(ctx context.Context) error
	Close() error
}

// NoopWriter discards all metrics.
type NoopWriter struct{}

func (NoopWriter) WritePoint(_ context.Context, _ Point) error { return nil }
func (NoopWriter) Flush(_ context.Context) error               { return nil }
func (NoopWriter) Close() error                                 { return nil }

var _ Writer = NoopWriter{}