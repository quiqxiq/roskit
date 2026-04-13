package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// GRETunnelSpec streams GRE tunnel interface status changes.
type GRETunnelSpec struct{}

func NewGRETunnelSpec() *GRETunnelSpec { return &GRETunnelSpec{} }

func (s *GRETunnelSpec) Command() []string {
	return []string{
		"/interface/gre/print",
		"=follow",
		"=.proplist=.id,name,remote-address,local-address,running,disabled,mtu,comment",
	}
}

func (s *GRETunnelSpec) Tag() string         { return "gre-tunnel" }
func (s *GRETunnelSpec) Measurement() string { return "gre_tunnel" }

func (s *GRETunnelSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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
		ID: id, Name: pairs["name"], Type: "gre",
		RemoteAddress: pairs["remote-address"], LocalAddress: pairs["local-address"],
		Running: mikrotik.ParseBool(pairs["running"]),
		Disabled: mikrotik.ParseBool(pairs["disabled"]),
		MTU: pairs["mtu"], Comment: pairs["comment"], Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: tunnel.ToTags(), Fields: tunnel.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: tunnel.ToCacheData(now),
	}, nil
}
