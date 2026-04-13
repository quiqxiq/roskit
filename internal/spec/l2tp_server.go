package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// L2TPServerSpec polls L2TP server enabled status.
// Uses =interval mode since /interface/l2tp-server/server does not support =follow.
type L2TPServerSpec struct{}

func NewL2TPServerSpec() *L2TPServerSpec { return &L2TPServerSpec{} }

func (s *L2TPServerSpec) Command() []string {
	return []string{
		"/interface/l2tp-server/server/print",
		"=interval=30s",
	}
}

func (s *L2TPServerSpec) Tag() string         { return "l2tp-server" }
func (s *L2TPServerSpec) Measurement() string { return "l2tp_server" }

func (s *L2TPServerSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
	pairs := sentence.Map
	if len(pairs) == 0 {
		return nil, nil
	}
	now := time.Now()

	fields := map[string]interface{}{
		"enabled":         pairs["enabled"],
		"authentication":  pairs["authentication"],
		"default_profile": pairs["default-profile"],
	}
	tags := map[string]string{
		"enabled": pairs["enabled"],
	}
	cacheData := map[string]string{
		"enabled": pairs["enabled"], "authentication": pairs["authentication"],
		"default_profile": pairs["default-profile"],
		"timestamp": now.Format(time.RFC3339),
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: tags, Fields: fields,
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), "server"),
		CacheData: cacheData,
	}, nil
}
