package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// QueueSimpleStatsSpec streams simple queue statistics at a fixed interval.
// Uses /queue/simple/print with =stats and =interval=1s for periodic snapshots
// of bandwidth usage, packet rates, and drop counts per queue rule.
type QueueSimpleStatsSpec struct{}

// NewQueueSimpleStatsSpec creates a new QueueSimpleStatsSpec.
func NewQueueSimpleStatsSpec() *QueueSimpleStatsSpec {
	return &QueueSimpleStatsSpec{}
}

// Command returns the API command for streaming simple queue statistics.
// Uses =stats for periodic counter snapshots (not event-driven),
// and =.proplist to limit returned attributes.
func (s *QueueSimpleStatsSpec) Command() []string {
	return []string{
		"/queue/simple/print",
		"=stats",
		"=interval=1s",
		"=.proplist=name,target,rate,packet-rate,queued-bytes,queued-packets,bytes,packets,dropped,total-rate,total-bytes,total-packets,total-dropped,total-queued-bytes",
	}
}

// Tag returns the unique identifier for this stream.
func (s *QueueSimpleStatsSpec) Tag() string {
	return "queue-simple"
}

// Measurement returns the InfluxDB measurement name.
func (s *QueueSimpleStatsSpec) Measurement() string {
	return "queue_simple_stats"
}

// Parse converts a raw API sentence into a QueueSimpleStats TelemetryEvent.
// Queue stats are periodic snapshots — no .dead events.
func (s *QueueSimpleStatsSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
	pairs := sentence.Map
	if len(pairs) == 0 {
		return nil, nil
	}

	name := pairs["name"]
	if name == "" {
		return nil, nil
	}

	now := time.Now()

	stats := domain.QueueSimpleStats{
		Name:             name,
		Target:           pairs["target"],
		Rate:             pairs["rate"],
		PacketRate:       pairs["packet-rate"],
		QueuedBytes:      pairs["queued-bytes"],
		QueuedPackets:    pairs["queued-packets"],
		Bytes:            pairs["bytes"],
		Packets:          pairs["packets"],
		Dropped:          pairs["dropped"],
		TotalRate:        pairs["total-rate"],
		TotalBytes:       pairs["total-bytes"],
		TotalPackets:     pairs["total-packets"],
		TotalDropped:     pairs["total-dropped"],
		TotalQueuedBytes: pairs["total-queued-bytes"],
		Timestamp:        now,
	}

	cacheKey := mikrotik.FormatCacheKey(routerID, s.Measurement(), name)

	return &domain.TelemetryEvent{
		RouterID:    routerID,
		Measurement: s.Measurement(),
		Tags:        stats.ToTags(),
		Fields:      stats.ToFields(),
		Timestamp:   now,
		Type:        domain.EventUpdate,
		CacheKey:    cacheKey,
		CacheData:   stats.ToCacheData(now),
	}, nil
}
