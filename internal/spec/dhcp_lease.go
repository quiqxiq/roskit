package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// DHCPLeaseSpec streams DHCP server lease changes in real-time.
// Uses /ip/dhcp-server/lease/print with follow to detect new leases,
// modifications, and expirations. The .dead=yes attribute is sent when a lease expires.
type DHCPLeaseSpec struct{}

// NewDHCPLeaseSpec creates a new DHCPLeaseSpec.
func NewDHCPLeaseSpec() *DHCPLeaseSpec {
	return &DHCPLeaseSpec{}
}

// Command returns the API command for streaming DHCP lease changes.
func (s *DHCPLeaseSpec) Command() []string {
	return []string{
		"/ip/dhcp-server/lease/print",
		"=follow",
		"=.proplist=.id,address,mac-address,client-id,server,lease-time,comment,disabled,block-access,rate-limit,routes,address-lists,dhcp-option,dhcp-option-set,always-broadcast",
	}
}

// Tag returns the unique identifier for this stream.
func (s *DHCPLeaseSpec) Tag() string {
	return "dhcp-lease"
}

// Measurement returns the InfluxDB measurement name.
func (s *DHCPLeaseSpec) Measurement() string {
	return "dhcp_lease"
}

// Parse converts a raw API sentence into a DHCPLease TelemetryEvent.
func (s *DHCPLeaseSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
	pairs := sentence.Map
	if len(pairs) == 0 {
		return nil, nil
	}

	id := pairs[".id"]
	if id == "" {
		return nil, nil
	}

	now := time.Now()

	// Check for dead entity (lease expired or removed).
	if mikrotik.IsDead(pairs) {
		return &domain.TelemetryEvent{
			RouterID:    routerID,
			Measurement: s.Measurement(),
			Type:        domain.EventDead,
			Timestamp:   now,
			CacheKey:    mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		}, nil
	}

	lease := domain.DHCPLease{
		ID:              id,
		Address:         pairs["address"],
		MacAddress:      pairs["mac-address"],
		ClientID:        pairs["client-id"],
		Server:          pairs["server"],
		LeaseTime:       pairs["lease-time"],
		Comment:         pairs["comment"],
		Disabled:        mikrotik.ParseBool(pairs["disabled"]),
		BlockAccess:     mikrotik.ParseBool(pairs["block-access"]),
		RateLimit:       pairs["rate-limit"],
		Routes:          pairs["routes"],
		AddressLists:    pairs["address-lists"],
		DHCPOption:      pairs["dhcp-option"],
		DHCPOptionSet:   pairs["dhcp-option-set"],
		AlwaysBroadcast: mikrotik.ParseBool(pairs["always-broadcast"]),
		Timestamp:       now,
	}

	return &domain.TelemetryEvent{
		RouterID:    routerID,
		Measurement: s.Measurement(),
		Tags:        lease.ToTags(),
		Fields:      lease.ToFields(),
		Timestamp:   now,
		Type:        domain.EventUpdate,
		CacheKey:    mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData:   lease.ToCacheData(now),
	}, nil
}
