package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// PPPSecretSpec streams PPP secret (user account) changes in real-time.
// Uses /ppp/secret/print with follow to detect account additions, modifications,
// and deletions. The .dead=yes attribute is sent when a secret is removed.
type PPPSecretSpec struct{}

// NewPPPSecretSpec creates a new PPPSecretSpec.
func NewPPPSecretSpec() *PPPSecretSpec {
	return &PPPSecretSpec{}
}

// Command returns the API command for streaming PPP secret changes.
func (s *PPPSecretSpec) Command() []string {
	return []string{
		"/ppp/secret/print",
		"=follow",
		"=.proplist=.id,name,service,caller-id,profile,local-address,remote-address,routes,limit-bytes-in,limit-bytes-out,disabled,comment",
	}
}

// Tag returns the unique identifier for this stream.
func (s *PPPSecretSpec) Tag() string {
	return "ppp-secret"
}

// Measurement returns the InfluxDB measurement name.
func (s *PPPSecretSpec) Measurement() string {
	return "ppp_secret"
}

// Parse converts a raw API sentence into a PPPSecret TelemetryEvent.
func (s *PPPSecretSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
	pairs := sentence.Map
	if len(pairs) == 0 {
		return nil, nil
	}

	id := pairs[".id"]
	if id == "" {
		return nil, nil
	}

	now := time.Now()

	// Check for dead entity (secret removed).
	if mikrotik.IsDead(pairs) {
		return &domain.TelemetryEvent{
			RouterID:    routerID,
			Measurement: s.Measurement(),
			Type:        domain.EventDead,
			Timestamp:   now,
			CacheKey:    mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		}, nil
	}

	secret := domain.PPPSecret{
		ID:            id,
		Name:          pairs["name"],
		Service:       pairs["service"],
		CallerID:      pairs["caller-id"],
		Profile:       pairs["profile"],
		LocalAddress:  pairs["local-address"],
		RemoteAddress: pairs["remote-address"],
		Routes:        pairs["routes"],
		LimitBytesIn:  mikrotik.ParseUint64(pairs["limit-bytes-in"]),
		LimitBytesOut: mikrotik.ParseUint64(pairs["limit-bytes-out"]),
		Disabled:      mikrotik.ParseBool(pairs["disabled"]),
		Comment:       pairs["comment"],
		Timestamp:     now,
	}

	return &domain.TelemetryEvent{
		RouterID:    routerID,
		Measurement: s.Measurement(),
		Tags:        secret.ToTags(),
		Fields:      secret.ToFields(),
		Timestamp:   now,
		Type:        domain.EventUpdate,
		CacheKey:    mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData:   secret.ToCacheData(now),
	}, nil
}
