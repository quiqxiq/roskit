package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// QueueTreeStatsSpec streams queue tree statistics at a fixed interval.
// Complements QueueSimpleStatsSpec for hierarchical QoS monitoring.
type QueueTreeStatsSpec struct{}

func NewQueueTreeStatsSpec() *QueueTreeStatsSpec { return &QueueTreeStatsSpec{} }

func (s *QueueTreeStatsSpec) Command() []string {
	return []string{
		"/queue/tree/print",
		"=stats",
		"=interval=1s",
		"=.proplist=name,parent,packet-mark,rate,packet-rate,queued-bytes,queued-packets,bytes,packets,dropped,max-limit,limit-at,burst-limit",
	}
}

func (s *QueueTreeStatsSpec) Tag() string         { return "queue-tree" }
func (s *QueueTreeStatsSpec) Measurement() string { return "queue_tree_stats" }

func (s *QueueTreeStatsSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
	pairs := sentence.Map
	if len(pairs) == 0 {
		return nil, nil
	}

	name := pairs["name"]
	if name == "" {
		return nil, nil
	}

	now := time.Now()

	stats := domain.QueueTreeStats{
		Name: name, Parent: pairs["parent"], PacketMark: pairs["packet-mark"],
		Rate: pairs["rate"], PacketRate: pairs["packet-rate"],
		QueuedBytes: pairs["queued-bytes"], QueuedPackets: pairs["queued-packets"],
		Bytes: pairs["bytes"], Packets: pairs["packets"], Dropped: pairs["dropped"],
		MaxLimit: pairs["max-limit"], LimitAt: pairs["limit-at"],
		BurstLimit: pairs["burst-limit"], Timestamp: now,
	}

	cacheKey := mikrotik.FormatCacheKey(routerID, s.Measurement(), name)

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: stats.ToTags(), Fields: stats.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  cacheKey,
		CacheData: stats.ToCacheData(now),
	}, nil
}
