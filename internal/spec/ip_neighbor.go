package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// IPNeighborSpec streams IP neighbor discovery (LLDP/CDP/MNDP) changes.
// Essential for NOC topology mapping — detects when neighbors appear/disappear.
type IPNeighborSpec struct{}

func NewIPNeighborSpec() *IPNeighborSpec { return &IPNeighborSpec{} }

func (s *IPNeighborSpec) Command() []string {
	return []string{
		"/ip/neighbor/print",
		"=follow",
		"=.proplist=.id,interface,address,mac-address,identity,platform,board,version,interface-name",
	}
}

func (s *IPNeighborSpec) Tag() string         { return "ip-neighbor" }
func (s *IPNeighborSpec) Measurement() string { return "ip_neighbor" }

func (s *IPNeighborSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	entry := domain.IPNeighborEntry{
		ID: id, Interface: pairs["interface"], Address: pairs["address"],
		MacAddress: pairs["mac-address"], Identity: pairs["identity"],
		Platform: pairs["platform"], Board: pairs["board"],
		Version: pairs["version"], InterfName: pairs["interface-name"],
		Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: entry.ToTags(), Fields: entry.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: entry.ToCacheData(now),
	}, nil
}
