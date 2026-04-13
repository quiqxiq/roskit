package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// NetwatchSpec streams Netwatch host availability status.
// Uses =follow+stats to get real-time status changes with counters.
type NetwatchSpec struct{}

func NewNetwatchSpec() *NetwatchSpec { return &NetwatchSpec{} }

func (s *NetwatchSpec) Command() []string {
	return []string{
		"/tool/netwatch/print",
		"=follow",
		"=.proplist=.id,host,status,interval,timeout,since,disabled,comment,type",
	}
}

func (s *NetwatchSpec) Tag() string         { return "netwatch" }
func (s *NetwatchSpec) Measurement() string { return "netwatch" }

func (s *NetwatchSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
	pairs := sentence.Map
	if len(pairs) == 0 {
		return nil, nil
	}
	id := pairs[".id"]
	if id == "" {
		return nil, nil
	}
	now := time.Now()

	if mikrotik.IsDead(pairs) {
		return &domain.TelemetryEvent{
			RouterID: routerID, Measurement: s.Measurement(),
			Type: domain.EventDead, Timestamp: now,
			CacheKey: mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		}, nil
	}

	entry := domain.NetwatchEntry{
		ID: id, Host: pairs["host"], Status: pairs["status"],
		Interval: pairs["interval"], Timeout: pairs["timeout"],
		Since: pairs["since"], Comment: pairs["comment"],
		Disabled: mikrotik.ParseBool(pairs["disabled"]),
		Type: pairs["type"], Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: entry.ToTags(), Fields: entry.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: entry.ToCacheData(now),
	}, nil
}
