package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// PPPProfileSpec streams PPP profile configuration changes in real-time.
// Uses /ppp/profile/print with follow to detect profile additions, modifications,
// and deletions. The .dead=yes attribute is sent when a profile is removed.
type PPPProfileSpec struct{}

// NewPPPProfileSpec creates a new PPPProfileSpec.
func NewPPPProfileSpec() *PPPProfileSpec {
	return &PPPProfileSpec{}
}

// Command returns the API command for streaming PPP profile changes.
func (s *PPPProfileSpec) Command() []string {
	return []string{
		"/ppp/profile/print",
		"=follow",
		"=.proplist=.id,name,local-address,remote-address,rate-limit,address-list,dns-server,on-up,on-down,session-timeout,idle-timeout,comment",
	}
}

// Tag returns the unique identifier for this stream.
func (s *PPPProfileSpec) Tag() string {
	return "ppp-profile"
}

// Measurement returns the InfluxDB measurement name.
func (s *PPPProfileSpec) Measurement() string {
	return "ppp_profile"
}

// Parse converts a raw API sentence into a PPPProfile TelemetryEvent.
func (s *PPPProfileSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
	pairs := sentence.Map
	if len(pairs) == 0 {
		return nil, nil
	}

	id := pairs[".id"]
	if id == "" {
		return nil, nil
	}

	now := time.Now()

	// Check for dead entity (profile removed).
	if mikrotik.IsDead(pairs) {
		return &domain.TelemetryEvent{
			RouterID:    routerID,
			Measurement: s.Measurement(),
			Type:        domain.EventDead,
			Timestamp:   now,
			CacheKey:    mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		}, nil
	}

	profile := domain.PPPProfile{
		ID:             id,
		Name:           pairs["name"],
		LocalAddress:   pairs["local-address"],
		RemoteAddress:  pairs["remote-address"],
		RateLimit:      pairs["rate-limit"],
		AddressList:    pairs["address-list"],
		DNSServer:      pairs["dns-server"],
		OnUp:           pairs["on-up"],
		OnDown:         pairs["on-down"],
		SessionTimeout: pairs["session-timeout"],
		IdleTimeout:    pairs["idle-timeout"],
		Comment:        pairs["comment"],
		Timestamp:      now,
	}

	return &domain.TelemetryEvent{
		RouterID:    routerID,
		Measurement: s.Measurement(),
		Tags:        profile.ToTags(),
		Fields:      profile.ToFields(),
		Timestamp:   now,
		Type:        domain.EventUpdate,
		CacheKey:    mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData:   profile.ToCacheData(now),
	}, nil
}
