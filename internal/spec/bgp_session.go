package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// BGPSessionSpec streams BGP session state changes in real-time.
// Critical for NOC: detects upstream/peer session drops immediately.
type BGPSessionSpec struct{}

func NewBGPSessionSpec() *BGPSessionSpec { return &BGPSessionSpec{} }

func (s *BGPSessionSpec) Command() []string {
	return []string{
		"/routing/bgp/session/print",
		"=follow",
		"=.proplist=.id,name,remote.address,remote.as,local.role,remote.role,state,uptime,prefix-count,established,disabled",
	}
}

func (s *BGPSessionSpec) Tag() string         { return "bgp-session" }
func (s *BGPSessionSpec) Measurement() string { return "bgp_session" }

func (s *BGPSessionSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	session := domain.BGPSession{
		ID: id, Name: pairs["name"],
		RemoteAddress: pairs["remote.address"], RemoteAS: pairs["remote.as"],
		LocalRole: pairs["local.role"], RemoteRole: pairs["remote.role"],
		State: pairs["state"], Uptime: pairs["uptime"],
		PrefixCount: pairs["prefix-count"], EstablishedCount: pairs["established"],
		Disabled: mikrotik.ParseBool(pairs["disabled"]), Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: session.ToTags(), Fields: session.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: session.ToCacheData(now),
	}, nil
}
