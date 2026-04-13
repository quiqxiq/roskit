package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// ARPTableSpec streams ARP table changes.
// Useful for tracking MAC-IP mappings and detecting ARP spoofing.
type ARPTableSpec struct{}

func NewARPTableSpec() *ARPTableSpec { return &ARPTableSpec{} }

func (s *ARPTableSpec) Command() []string {
	return []string{
		"/ip/arp/print",
		"=follow",
		"=.proplist=.id,address,mac-address,interface,published,disabled,comment",
	}
}

func (s *ARPTableSpec) Tag() string         { return "arp-table" }
func (s *ARPTableSpec) Measurement() string { return "arp_table" }

func (s *ARPTableSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	entry := domain.ARPEntry{
		ID: id, Address: pairs["address"], MacAddress: pairs["mac-address"],
		Interface: pairs["interface"], Published: mikrotik.ParseBool(pairs["published"]),
		Disabled: mikrotik.ParseBool(pairs["disabled"]),
		Comment: pairs["comment"], Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: entry.ToTags(), Fields: entry.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: entry.ToCacheData(now),
	}, nil
}
