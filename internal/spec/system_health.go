package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// SystemHealthSpec polls hardware health sensors (temperature, voltage, fan speed).
// Uses =interval mode since /system/health does not support =follow.
type SystemHealthSpec struct{}

func NewSystemHealthSpec() *SystemHealthSpec { return &SystemHealthSpec{} }

func (s *SystemHealthSpec) Command() []string {
	return []string{
		"/system/health/print",
		"=interval=10s",
		"=.proplist=name,value,type",
	}
}

func (s *SystemHealthSpec) Tag() string         { return "system-health" }
func (s *SystemHealthSpec) Measurement() string { return "system_health" }

func (s *SystemHealthSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
	pairs := sentence.Map
	if len(pairs) == 0 {
		return nil, nil
	}

	name := pairs["name"]
	if name == "" {
		return nil, nil
	}

	now := time.Now()

	health := domain.SystemHealth{
		Name: name, Value: pairs["value"], Type: pairs["type"],
		Timestamp: now,
	}

	cacheKey := mikrotik.FormatCacheKey(routerID, s.Measurement(), name)

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: health.ToTags(), Fields: health.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  cacheKey,
		CacheData: health.ToCacheData(now),
	}, nil
}
