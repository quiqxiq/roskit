package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// UserActiveSpec streams active user sessions on the router.
// Detects who is logged in — important for audit and security.
type UserActiveSpec struct{}

func NewUserActiveSpec() *UserActiveSpec { return &UserActiveSpec{} }

func (s *UserActiveSpec) Command() []string {
	return []string{
		"/user/active/print",
		"=follow",
		"=.proplist=.id,name,address,via,when,group",
	}
}

func (s *UserActiveSpec) Tag() string         { return "user-active" }
func (s *UserActiveSpec) Measurement() string { return "user_active" }

func (s *UserActiveSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	user := domain.UserActive{
		ID: id, Name: pairs["name"], Address: pairs["address"],
		Via: pairs["via"], When: pairs["when"], Group: pairs["group"],
		Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: user.ToTags(), Fields: user.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: user.ToCacheData(now),
	}, nil
}
