package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// WireguardPeersSpec streams WireGuard peer state and traffic.
type WireguardPeersSpec struct{}

func NewWireguardPeersSpec() *WireguardPeersSpec { return &WireguardPeersSpec{} }

func (s *WireguardPeersSpec) Command() []string {
	return []string{
		"/interface/wireguard/peers/print",
		"=follow",
		"=.proplist=.id,interface,public-key,endpoint-address,endpoint-port,allowed-address,last-handshake,rx,tx,disabled,comment",
	}
}

func (s *WireguardPeersSpec) Tag() string         { return "wireguard-peers" }
func (s *WireguardPeersSpec) Measurement() string { return "wireguard_peers" }

func (s *WireguardPeersSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	peer := domain.WireguardPeer{
		ID: id, Interface: pairs["interface"], PublicKey: pairs["public-key"],
		EndpointAddr: pairs["endpoint-address"], EndpointPort: pairs["endpoint-port"],
		AllowedAddress: pairs["allowed-address"], LastHandshake: pairs["last-handshake"],
		Rx: pairs["rx"], Tx: pairs["tx"],
		Disabled: mikrotik.ParseBool(pairs["disabled"]),
		Comment:  pairs["comment"], Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: peer.ToTags(), Fields: peer.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: peer.ToCacheData(now),
	}, nil
}
