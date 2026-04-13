package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// BridgeHostSpec streams bridge MAC address table changes.
type BridgeHostSpec struct{}

func NewBridgeHostSpec() *BridgeHostSpec { return &BridgeHostSpec{} }

func (s *BridgeHostSpec) Command() []string {
	return []string{
		"/interface/bridge/host/print",
		"=follow",
		"=.proplist=.id,bridge,interface,mac-address,vid,on-interface,local,disabled",
	}
}

func (s *BridgeHostSpec) Tag() string         { return "bridge-host" }
func (s *BridgeHostSpec) Measurement() string { return "bridge_host" }

func (s *BridgeHostSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	host := domain.BridgeHost{
		ID: id, Bridge: pairs["bridge"], Interface: pairs["interface"],
		MacAddr: pairs["mac-address"], VID: pairs["vid"],
		OnLocal:  mikrotik.ParseBool(pairs["local"]),
		Disabled: mikrotik.ParseBool(pairs["disabled"]), Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: host.ToTags(), Fields: host.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: host.ToCacheData(now),
	}, nil
}
