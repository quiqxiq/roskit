package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// CAPsMANInterfaceSpec streams CAPsMAN managed AP interface status.
type CAPsMANInterfaceSpec struct{}

func NewCAPsMANInterfaceSpec() *CAPsMANInterfaceSpec { return &CAPsMANInterfaceSpec{} }

func (s *CAPsMANInterfaceSpec) Command() []string {
	return []string{
		"/caps-man/interface/print",
		"=follow",
		"=.proplist=.id,name,radio-name,radio-mac,master-interface,current-state,bound,inactive,disabled,comment",
	}
}

func (s *CAPsMANInterfaceSpec) Tag() string         { return "capsman-iface" }
func (s *CAPsMANInterfaceSpec) Measurement() string { return "capsman_interface" }

func (s *CAPsMANInterfaceSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	iface := domain.CAPsMANInterface{
		ID: id, Name: pairs["name"], RadioName: pairs["radio-name"],
		RadioMAC: pairs["radio-mac"], MasterIface: pairs["master-interface"],
		CurrentState: pairs["current-state"],
		Bound: mikrotik.ParseBool(pairs["bound"]),
		Inactive: mikrotik.ParseBool(pairs["inactive"]),
		Disabled: mikrotik.ParseBool(pairs["disabled"]),
		Comment: pairs["comment"], Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: iface.ToTags(), Fields: iface.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: iface.ToCacheData(now),
	}, nil
}
