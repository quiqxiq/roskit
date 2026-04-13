package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// DNSCacheSpec streams DNS cache entry changes.
type DNSCacheSpec struct{}

func NewDNSCacheSpec() *DNSCacheSpec { return &DNSCacheSpec{} }

func (s *DNSCacheSpec) Command() []string {
	return []string{
		"/ip/dns/cache/print",
		"=follow",
		"=.proplist=.id,name,address,ttl,type",
	}
}

func (s *DNSCacheSpec) Tag() string         { return "dns-cache" }
func (s *DNSCacheSpec) Measurement() string { return "dns_cache" }

func (s *DNSCacheSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	entry := domain.DNSCacheEntry{
		ID: id, Name: pairs["name"], Address: pairs["address"],
		TTL: pairs["ttl"], Type: pairs["type"], Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: entry.ToTags(), Fields: entry.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: entry.ToCacheData(now),
	}, nil
}
