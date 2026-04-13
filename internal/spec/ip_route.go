package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// IPRouteSpec streams IP routing table changes.
// Detects route flapping, new routes, and route removals.
type IPRouteSpec struct{}

func NewIPRouteSpec() *IPRouteSpec { return &IPRouteSpec{} }

func (s *IPRouteSpec) Command() []string {
	return []string{
		"/ip/route/print",
		"=follow",
		"=.proplist=.id,dst-address,gateway,distance,routing-table,pref-src,scope,target-scope,disabled,comment",
	}
}

func (s *IPRouteSpec) Tag() string         { return "ip-route" }
func (s *IPRouteSpec) Measurement() string { return "ip_route" }

func (s *IPRouteSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	route := domain.IPRoute{
		ID: id, DstAddress: pairs["dst-address"], Gateway: pairs["gateway"],
		Distance: pairs["distance"], RoutingTable: pairs["routing-table"],
		PrefSrc: pairs["pref-src"], Scope: pairs["scope"],
		TargetScope: pairs["target-scope"],
		Disabled: mikrotik.ParseBool(pairs["disabled"]),
		Comment: pairs["comment"], Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: route.ToTags(), Fields: route.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: route.ToCacheData(now),
	}, nil
}
