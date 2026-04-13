package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// IPPoolUsedSpec streams used IP pool addresses.
// Critical for ISP: detects pool exhaustion before subscribers fail to connect.
type IPPoolUsedSpec struct{}

func NewIPPoolUsedSpec() *IPPoolUsedSpec { return &IPPoolUsedSpec{} }

func (s *IPPoolUsedSpec) Command() []string {
	return []string{
		"/ip/pool/used/print",
		"=follow",
		"=.proplist=.id,pool,address,owner,info",
	}
}

func (s *IPPoolUsedSpec) Tag() string         { return "ip-pool-used" }
func (s *IPPoolUsedSpec) Measurement() string { return "ip_pool_used" }

func (s *IPPoolUsedSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	used := domain.IPPoolUsed{
		ID: id, Pool: pairs["pool"], Address: pairs["address"],
		Owner: pairs["owner"], Info: pairs["info"], Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: used.ToTags(), Fields: used.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: used.ToCacheData(now),
	}, nil
}
