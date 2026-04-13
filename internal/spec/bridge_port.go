package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// BridgePortSpec streams bridge port STP state and configuration changes.
type BridgePortSpec struct{}

func NewBridgePortSpec() *BridgePortSpec { return &BridgePortSpec{} }

func (s *BridgePortSpec) Command() []string {
	return []string{
		"/interface/bridge/port/print",
		"=follow",
		"=.proplist=.id,bridge,interface,role,status,port-number,priority,edge,learn,disabled,comment",
	}
}

func (s *BridgePortSpec) Tag() string         { return "bridge-port" }
func (s *BridgePortSpec) Measurement() string { return "bridge_port" }

func (s *BridgePortSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	port := domain.BridgePort{
		ID: id, Bridge: pairs["bridge"], Interface: pairs["interface"],
		Role: pairs["role"], Status: pairs["status"],
		PortNum: pairs["port-number"], Priority: pairs["priority"],
		Edge: pairs["edge"], Learning: mikrotik.ParseBool(pairs["learn"]),
		Disabled: mikrotik.ParseBool(pairs["disabled"]),
		Comment:  pairs["comment"], Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: port.ToTags(), Fields: port.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: port.ToCacheData(now),
	}, nil
}
