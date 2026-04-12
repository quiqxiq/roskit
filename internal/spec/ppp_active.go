package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// PPPActiveSpec streams active PPPoE/PPTP/L2TP sessions in real-time.
// Uses /ppp/active/print with follow to detect new connections and disconnections.
// The .dead=yes attribute is sent when a session terminates.
type PPPActiveSpec struct{}

// NewPPPActiveSpec creates a new PPPActiveSpec.
func NewPPPActiveSpec() *PPPActiveSpec {
	return &PPPActiveSpec{}
}

// Command returns the API command for streaming PPP active sessions.
// Uses =follow for continuous event-driven updates,
// and =.proplist to limit returned attributes for router CPU efficiency.
func (s *PPPActiveSpec) Command() []string {
	return []string{
		"/ppp/active/print",
		"=follow",
		"=.proplist=.id,name,service,caller-id,address,uptime,encoding",
	}
}

// Tag returns the unique identifier for this stream.
func (s *PPPActiveSpec) Tag() string {
	return "ppp-active"
}

// Measurement returns the InfluxDB measurement name.
func (s *PPPActiveSpec) Measurement() string {
	return "ppp_active"
}

// Parse converts a raw API sentence into a PPPActiveSession TelemetryEvent.
// When a session disconnects, the router sends .dead=yes, triggering cache deletion.
func (s *PPPActiveSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
	pairs := sentence.Map
	if len(pairs) == 0 {
		return nil, nil
	}

	id := pairs[".id"]
	if id == "" {
		return nil, nil
	}

	now := time.Now()

	// Check for dead entity (session disconnected).
	if mikrotik.IsDead(pairs) {
		return &domain.TelemetryEvent{
			RouterID:    routerID,
			Measurement: s.Measurement(),
			Type:        domain.EventDead,
			Timestamp:   now,
			CacheKey:    mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		}, nil
	}

	session := domain.PPPActiveSession{
		ID:        id,
		Name:      pairs["name"],
		Service:   pairs["service"],
		CallerID:  pairs["caller-id"],
		Address:   pairs["address"],
		Uptime:    pairs["uptime"],
		Encoding:  pairs["encoding"],
		Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID:    routerID,
		Measurement: s.Measurement(),
		Tags:        session.ToTags(),
		Fields:      session.ToFields(),
		Timestamp:   now,
		Type:        domain.EventUpdate,
		CacheKey:    mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: map[string]string{
			"id":        session.ID,
			"name":      session.Name,
			"service":   session.Service,
			"caller_id": session.CallerID,
			"address":   session.Address,
			"uptime":    session.Uptime,
			"encoding":  session.Encoding,
			"timestamp": now.Format(time.RFC3339),
		},
	}, nil
}
