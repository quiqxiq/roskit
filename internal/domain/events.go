package domain

import "time"

// EventType represents the type of telemetry event.
type EventType int

const (
	// EventUpdate indicates a new or updated data point.
	EventUpdate EventType = iota
	// EventDead indicates an entity was removed from the router (e.g., user logout, interface removed).
	// This is triggered by the .dead=yes attribute in MikroTik API responses.
	EventDead
)

// String returns a human-readable representation of the event type.
func (e EventType) String() string {
	switch e {
	case EventUpdate:
		return "update"
	case EventDead:
		return "dead"
	default:
		return "unknown"
	}
}

// TelemetryEvent wraps a domain entity with metadata about the router and event context.
// It is the standard data unit flowing through the usecase layer.
type TelemetryEvent struct {
	// RouterID identifies which router produced this event.
	RouterID string
	// Measurement is the InfluxDB measurement name (e.g., "interface_stats").
	Measurement string
	// Tags are the InfluxDB tag key-value pairs for this data point.
	Tags map[string]string
	// Fields are the InfluxDB field key-value pairs for this data point.
	Fields map[string]interface{}
	// Timestamp is the time at which this event was captured.
	Timestamp time.Time
	// Type indicates whether this is an update or a dead (removal) event.
	Type EventType
	// CacheKey is the Redis key used to store the latest snapshot of this entity.
	// Format: "roskit:{routerID}:{measurement}:{entity_identifier}"
	CacheKey string
	// CacheData holds the JSON-serializable snapshot for Redis HSET.
	// Only populated for EventUpdate; nil for EventDead.
	CacheData map[string]string
}
