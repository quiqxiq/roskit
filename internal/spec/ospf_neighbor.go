package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// OSPFNeighborSpec streams OSPF neighbor adjacency state changes.
// Critical for NOC: detects routing topology changes instantly.
type OSPFNeighborSpec struct{}

func NewOSPFNeighborSpec() *OSPFNeighborSpec { return &OSPFNeighborSpec{} }

func (s *OSPFNeighborSpec) Command() []string {
	return []string{
		"/routing/ospf/neighbor/print",
		"=follow",
		"=.proplist=.id,instance,router-id,address,interface,priority,state,state-changes,adjacency",
	}
}

func (s *OSPFNeighborSpec) Tag() string         { return "ospf-neighbor" }
func (s *OSPFNeighborSpec) Measurement() string { return "ospf_neighbor" }

func (s *OSPFNeighborSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	neighbor := domain.OSPFNeighbor{
		ID: id, Instance: pairs["instance"], RouterID: pairs["router-id"],
		Address: pairs["address"], Interface: pairs["interface"],
		Priority: pairs["priority"], State: pairs["state"],
		StateChanges: pairs["state-changes"], Adjacency: pairs["adjacency"],
		Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: neighbor.ToTags(), Fields: neighbor.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: neighbor.ToCacheData(now),
	}, nil
}
