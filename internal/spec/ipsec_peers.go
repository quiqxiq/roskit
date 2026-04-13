package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// IPsecActivePeersSpec streams active IPsec peer status with traffic counters.
// Critical for NOC: detects site-to-site VPN tunnel failures immediately.
type IPsecActivePeersSpec struct{}

func NewIPsecActivePeersSpec() *IPsecActivePeersSpec { return &IPsecActivePeersSpec{} }

func (s *IPsecActivePeersSpec) Command() []string {
	return []string{
		"/ip/ipsec/active-peers/print",
		"=follow",
		"=.proplist=.id,remote-address,local-address,state,side,uptime,rx-bytes,tx-bytes,rx-packets,tx-packets,dynamic-address,responder",
	}
}

func (s *IPsecActivePeersSpec) Tag() string         { return "ipsec-peers" }
func (s *IPsecActivePeersSpec) Measurement() string { return "ipsec_active_peers" }

func (s *IPsecActivePeersSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	peer := domain.IPsecActivePeer{
		ID: id, RemoteAddress: pairs["remote-address"], LocalAddress: pairs["local-address"],
		State: pairs["state"], Side: pairs["side"], Uptime: pairs["uptime"],
		RxBytes: pairs["rx-bytes"], TxBytes: pairs["tx-bytes"],
		RxPackets: pairs["rx-packets"], TxPackets: pairs["tx-packets"],
		DynAddr: pairs["dynamic-address"], Responder: mikrotik.ParseBool(pairs["responder"]),
		Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: peer.ToTags(), Fields: peer.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: peer.ToCacheData(now),
	}, nil
}
