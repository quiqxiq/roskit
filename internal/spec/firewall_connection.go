package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// FirewallConnectionSpec streams active firewall connection tracking.
// Monitors conntrack entries — useful for connection count alerts and DDoS detection.
type FirewallConnectionSpec struct{}

func NewFirewallConnectionSpec() *FirewallConnectionSpec { return &FirewallConnectionSpec{} }

func (s *FirewallConnectionSpec) Command() []string {
	return []string{
		"/ip/firewall/connection/print",
		"=follow",
		"=.proplist=.id,protocol,src-address,dst-address,reply-src-address,reply-dst-address,tcp-state,timeout,assured",
	}
}

func (s *FirewallConnectionSpec) Tag() string         { return "firewall-connection" }
func (s *FirewallConnectionSpec) Measurement() string { return "firewall_connection" }

func (s *FirewallConnectionSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	conn := domain.FirewallConnection{
		ID: id, Protocol: pairs["protocol"],
		SrcAddress: pairs["src-address"], DstAddress: pairs["dst-address"],
		ReplySrc: pairs["reply-src-address"], ReplyDst: pairs["reply-dst-address"],
		TCPState: pairs["tcp-state"], Timeout: pairs["timeout"],
		Assured: mikrotik.ParseBool(pairs["assured"]), Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: conn.ToTags(), Fields: conn.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: conn.ToCacheData(now),
	}, nil
}
