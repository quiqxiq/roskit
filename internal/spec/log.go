package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// LogSpec streams router log entries in real-time.
// Uses /log/print with follow to capture all new log messages as they appear.
// Log entries are immutable — they are appended but never updated or deleted.
type LogSpec struct{}

// NewLogSpec creates a new LogSpec.
func NewLogSpec() *LogSpec {
	return &LogSpec{}
}

// Command returns the API command for streaming log entries.
// Uses =follow to get new log entries as they appear,
// and =.proplist to limit returned attributes.
func (s *LogSpec) Command() []string {
	return []string{
		"/log/print",
		"=follow",
		"=.proplist=.id,time,topics,message",
	}
}

// Tag returns the unique identifier for this stream.
func (s *LogSpec) Tag() string {
	return "log"
}

// Measurement returns the InfluxDB measurement name.
func (s *LogSpec) Measurement() string {
	return "router_log"
}

// Parse converts a raw API sentence into a LogEntry TelemetryEvent.
// Log entries don't have .dead events since logs are append-only.
func (s *LogSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
	pairs := sentence.Map
	if len(pairs) == 0 {
		return nil, nil
	}

	id := pairs[".id"]
	if id == "" {
		return nil, nil
	}

	now := time.Now()

	// Log entries are append-only, but follow can send .dead on log rotation.
	if mikrotik.IsDead(pairs) {
		return &domain.TelemetryEvent{
			RouterID:    routerID,
			Measurement: s.Measurement(),
			Type:        domain.EventDead,
			Timestamp:   now,
			CacheKey:    mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		}, nil
	}

	entry := domain.LogEntry{
		ID:        id,
		Time:      pairs["time"],
		Topics:    pairs["topics"],
		Message:   pairs["message"],
		Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID:    routerID,
		Measurement: s.Measurement(),
		Tags:        entry.ToTags(),
		Fields:      entry.ToFields(),
		Timestamp:   now,
		Type:        domain.EventUpdate,
		CacheKey:    mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData:   entry.ToCacheData(now),
	}, nil
}
