package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// IPAddressSpec streams IP address assignment changes.
// Detects when interfaces gain or lose IP addresses.
type IPAddressSpec struct{}

func NewIPAddressSpec() *IPAddressSpec { return &IPAddressSpec{} }

func (s *IPAddressSpec) Command() []string {
	return []string{
		"/ip/address/print",
		"=follow",
		"=.proplist=.id,address,network,interface,disabled,comment",
	}
}

func (s *IPAddressSpec) Tag() string         { return "ip-address" }
func (s *IPAddressSpec) Measurement() string { return "ip_address" }

func (s *IPAddressSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	addr := domain.IPAddress{
		ID: id, Address: pairs["address"], Network: pairs["network"],
		Interface: pairs["interface"], Disabled: mikrotik.ParseBool(pairs["disabled"]),
		Comment: pairs["comment"], Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: addr.ToTags(), Fields: addr.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: addr.ToCacheData(now),
	}, nil
}
