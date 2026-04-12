// Package repository defines the interfaces for data persistence and messaging.
// These interfaces follow the Dependency Inversion Principle — the business logic
// depends on abstractions, not concrete implementations.
package repository

import (
	"context"
	"time"
)

// TimeSeriesWriter defines the contract for writing time-series metrics.
// Implementations include InfluxDB, but the interface allows easy swapping.
type TimeSeriesWriter interface {
	// WritePoint writes a single data point to the time-series database.
	// measurement is the InfluxDB measurement name.
	// tags are indexed metadata (e.g., router ID, interface name).
	// fields are the actual metric values.
	// ts is the timestamp for this data point.
	WritePoint(ctx context.Context, measurement string, tags map[string]string, fields map[string]interface{}, ts time.Time) error

	// Flush forces all buffered points to be written immediately.
	// Should be called during graceful shutdown.
	Flush(ctx context.Context) error

	// Close releases all resources held by the writer.
	Close() error
}

// CacheRepository defines the contract for snapshot caching.
// Each telemetry entity's latest state is stored for instant retrieval
// without querying the historical time-series database.
type CacheRepository interface {
	// SetSnapshot stores the latest state of a telemetry entity.
	// key is the unique cache key (e.g., "roskit:core-01:interface_stats:ether1").
	// data is a map of field names to their string values, stored via HSET.
	SetSnapshot(ctx context.Context, key string, data map[string]string) error

	// GetSnapshot retrieves the latest cached state of a telemetry entity.
	// Returns all fields stored at the given key.
	GetSnapshot(ctx context.Context, key string) (map[string]string, error)

	// DeleteSnapshot removes a cached entity.
	// Called when .dead=yes is received, indicating the entity no longer exists on the router.
	DeleteSnapshot(ctx context.Context, key string) error

	// Close releases all resources held by the cache.
	Close() error
}

// PubSubPublisher defines the contract for real-time event distribution.
// Every telemetry update is published so external subscribers can react instantly.
type PubSubPublisher interface {
	// Publish sends a message to the specified channel.
	// Subscribers listening on this channel will receive the message in real-time.
	Publish(ctx context.Context, channel string, message []byte) error

	// Close releases all resources held by the publisher.
	Close() error
}
