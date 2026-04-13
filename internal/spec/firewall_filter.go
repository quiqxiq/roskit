package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// FirewallFilterSpec streams firewall filter rule changes and hit counters.
// Uses =stats mode for real-time bytes/packets counters per rule.
type FirewallFilterSpec struct{}

func NewFirewallFilterSpec() *FirewallFilterSpec { return &FirewallFilterSpec{} }

func (s *FirewallFilterSpec) Command() []string {
	return []string{
		"/ip/firewall/filter/print",
		"=stats",
		"=interval=5s",
		"=.proplist=.id,chain,action,comment,bytes,packets,disabled,src-address,dst-address,protocol,dst-port,src-port,in-interface,out-interface,connection-state",
	}
}

func (s *FirewallFilterSpec) Tag() string         { return "firewall-filter" }
func (s *FirewallFilterSpec) Measurement() string { return "firewall_filter" }

func (s *FirewallFilterSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	rule := domain.FirewallRule{
		ID: id, Chain: pairs["chain"], Action: pairs["action"], Comment: pairs["comment"],
		Bytes: mikrotik.ParseUint64(pairs["bytes"]), Packets: mikrotik.ParseUint64(pairs["packets"]),
		Disabled: mikrotik.ParseBool(pairs["disabled"]), SrcAddress: pairs["src-address"],
		DstAddress: pairs["dst-address"], Protocol: pairs["protocol"],
		DstPort: pairs["dst-port"], SrcPort: pairs["src-port"],
		InIface: pairs["in-interface"], OutIface: pairs["out-interface"],
		ConnState: pairs["connection-state"], Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: rule.ToTags(), Fields: rule.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: rule.ToCacheData(now),
	}, nil
}
