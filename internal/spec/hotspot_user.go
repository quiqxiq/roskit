package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// HotspotUserSpec streams all configured hotspot users.
// This is used for calculating inactive hotspot users.
type HotspotUserSpec struct{}

func NewHotspotUserSpec() *HotspotUserSpec { return &HotspotUserSpec{} }

func (s *HotspotUserSpec) Command() []string {
	return []string{
		"/ip/hotspot/user/print",
		"=follow",
		"=.proplist=.id,server,name,profile,mac-address,limit-uptime,limit-bytes-total,disabled,comment",
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
			RouterID: routerID, Measurement: s.Measurement(),
			Type: domain.EventDead, Timestamp: now,
			CacheKey: mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		}, nil
	}

	tags := map[string]string{
		"server": pairs["server"],
		"name":   pairs["name"],
	}
	fields := map[string]interface{}{
		"profile":           pairs["profile"],
		"mac_address":       pairs["mac-address"],
		"limit_uptime":      pairs["limit-uptime"],
		"limit_bytes_total": pairs["limit-bytes-total"],
		"disabled":          mikrotik.ParseBool(pairs["disabled"]),
		"comment":           pairs["comment"],
	}
	cacheData := map[string]string{
		"id": id, "server": pairs["server"], "name": pairs["name"],
		"profile": pairs["profile"], "mac_address": pairs["mac-address"],
		"limit_uptime": pairs["limit-uptime"], "limit_bytes_total": pairs["limit-bytes-total"],
		"disabled": pairs["disabled"], "comment": pairs["comment"],
		"timestamp": now.Format(time.RFC3339),
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: tags, Fields: fields,
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: cacheData,
	}, nil
}
