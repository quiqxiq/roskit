package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// EoIPTunnelSpec streams EoIP tunnel interface status changes.
type EoIPTunnelSpec struct{}

func NewEoIPTunnelSpec() *EoIPTunnelSpec { return &EoIPTunnelSpec{} }

func (s *EoIPTunnelSpec) Command() []string {
	return []string{
		"/interface/eoip/print",
		"=follow",
		"=.proplist=.id,name,remote-address,local-address,running,disabled,mtu,comment",
	}
}

func (s *EoIPTunnelSpec) Tag() string         { return "eoip-tunnel" }
func (s *EoIPTunnelSpec) Measurement() string { return "eoip_tunnel" }

func (s *EoIPTunnelSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	tunnel := domain.TunnelInterface{
		ID: id, Name: pairs["name"], Type: "eoip",
		RemoteAddress: pairs["remote-address"], LocalAddress: pairs["local-address"],
		Running:  mikrotik.ParseBool(pairs["running"]),
		Disabled: mikrotik.ParseBool(pairs["disabled"]),
		MTU:      pairs["mtu"], Comment: pairs["comment"], Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: tunnel.ToTags(), Fields: tunnel.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: tunnel.ToCacheData(now),
	}, nil
}
