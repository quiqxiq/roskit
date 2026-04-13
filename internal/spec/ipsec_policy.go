package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// IPsecPolicySpec streams IPsec policy changes.
type IPsecPolicySpec struct{}

func NewIPsecPolicySpec() *IPsecPolicySpec { return &IPsecPolicySpec{} }

func (s *IPsecPolicySpec) Command() []string {
	return []string{
		"/ip/ipsec/policy/print",
		"=follow",
		"=.proplist=.id,peer,src-address,dst-address,protocol,action,level,disabled,comment",
	}
}

func (s *IPsecPolicySpec) Tag() string         { return "ipsec-policy" }
func (s *IPsecPolicySpec) Measurement() string { return "ipsec_policy" }

func (s *IPsecPolicySpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	policy := domain.IPsecPolicy{
		ID: id, Peer: pairs["peer"], SrcAddr: pairs["src-address"],
		DstAddr: pairs["dst-address"], Protocol: pairs["protocol"],
		Action: pairs["action"], Level: pairs["level"],
		Disabled: mikrotik.ParseBool(pairs["disabled"]),
		Comment:  pairs["comment"], Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: policy.ToTags(), Fields: policy.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: policy.ToCacheData(now),
	}, nil
}
