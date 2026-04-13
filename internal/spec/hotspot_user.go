package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// HotspotUserSpec streams configured hotspot users.
type HotspotUserSpec struct{}

func NewHotspotUserSpec() *HotspotUserSpec { return &HotspotUserSpec{} }

func (s *HotspotUserSpec) Command() []string {
	return []string{
		"/ip/hotspot/user/print",
		"=follow",
		"=.proplist=.id,server,name,profile,mac-address,uptime,bytes-in,bytes-out,disabled,comment",
	}
}

func (s *HotspotUserSpec) Tag() string         { return "hotspot-user" }
func (s *HotspotUserSpec) Measurement() string { return "hotspot_user" }

func (s *HotspotUserSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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
			RouterID:    routerID,
			Measurement: s.Measurement(),
			Type:        domain.EventDead,
			Timestamp:   now,
			CacheKey:    mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		}, nil
	}

	user := domain.HotspotUser{
		ID:        id,
		Server:    pairs["server"],
		Name:      pairs["name"],
		Profile:   pairs["profile"],
		MacAddr:   pairs["mac-address"],
		Uptime:    pairs["uptime"],
		BytesIn:   pairs["bytes-in"],
		BytesOut:  pairs["bytes-out"],
		Disabled:  mikrotik.ParseBool(pairs["disabled"]),
		Comment:   pairs["comment"],
		Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID:    routerID,
		Measurement: s.Measurement(),
		Tags:        user.ToTags(),
		Fields:      user.ToFields(),
		Timestamp:   now,
		Type:        domain.EventUpdate,
		CacheKey:    mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData:   user.ToCacheData(now),
	}, nil
}
