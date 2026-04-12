package spec

import (
	"fmt"
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// HotspotActiveSpec streams active hotspot user sessions in real-time.
// Uses /ip/hotspot/active/print with follow to detect new logins and logouts.
// The .dead=yes attribute is sent when a user disconnects.
type HotspotActiveSpec struct{}

// NewHotspotActiveSpec creates a new HotspotActiveSpec.
func NewHotspotActiveSpec() *HotspotActiveSpec {
	return &HotspotActiveSpec{}
}

// Command returns the API command for streaming hotspot active users.
// Uses =follow for continuous event-driven updates (new sessions and changes),
// and =.proplist to limit returned attributes.
func (s *HotspotActiveSpec) Command() []string {
	return []string{
		"/ip/hotspot/active/print",
		"=follow",
		"=.proplist=.id,user,address,mac-address,uptime,bytes-in,bytes-out",
	}
}

// Tag returns the unique identifier for this stream.
func (s *HotspotActiveSpec) Tag() string {
	return "hotspot-active"
}

// Measurement returns the InfluxDB measurement name.
func (s *HotspotActiveSpec) Measurement() string {
	return "hotspot_active"
}

// Parse converts a raw API sentence into a HotspotActiveUser TelemetryEvent.
// When a user logs out, the router sends .dead=yes, which triggers cache deletion.
func (s *HotspotActiveSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
	pairs := sentence.Map
	if len(pairs) == 0 {
		return nil, nil
	}

	id := pairs[".id"]
	if id == "" {
		return nil, nil
	}

	now := time.Now()

	// Check for dead entity (user logged out).
	if mikrotik.IsDead(pairs) {
		return &domain.TelemetryEvent{
			RouterID:    routerID,
			Measurement: s.Measurement(),
			Type:        domain.EventDead,
			Timestamp:   now,
			CacheKey:    mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		}, nil
	}

	user := domain.HotspotActiveUser{
		ID:         id,
		User:       pairs["user"],
		Address:    pairs["address"],
		MacAddress: pairs["mac-address"],
		Uptime:     pairs["uptime"],
		BytesIn:    mikrotik.ParseUint64(pairs["bytes-in"]),
		BytesOut:   mikrotik.ParseUint64(pairs["bytes-out"]),
		Timestamp:  now,
	}

	return &domain.TelemetryEvent{
		RouterID:    routerID,
		Measurement: s.Measurement(),
		Tags:        user.ToTags(),
		Fields:      user.ToFields(),
		Timestamp:   now,
		Type:        domain.EventUpdate,
		CacheKey:    mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: map[string]string{
			"id":          user.ID,
			"user":        user.User,
			"address":     user.Address,
			"mac_address": user.MacAddress,
			"uptime":      user.Uptime,
			"bytes_in":    fmt.Sprintf("%d", user.BytesIn),
			"bytes_out":   fmt.Sprintf("%d", user.BytesOut),
			"timestamp":   now.Format(time.RFC3339),
		},
	}, nil
}
