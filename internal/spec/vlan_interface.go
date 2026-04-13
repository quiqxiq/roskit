package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// VLANInterfaceSpec streams VLAN interface configuration changes.
type VLANInterfaceSpec struct{}

func NewVLANInterfaceSpec() *VLANInterfaceSpec { return &VLANInterfaceSpec{} }

func (s *VLANInterfaceSpec) Command() []string {
	return []string{
		"/interface/vlan/print",
		"=follow",
		"=.proplist=.id,name,vlan-id,interface,mtu,running,disabled,comment",
	}
}

func (s *VLANInterfaceSpec) Tag() string         { return "vlan-iface" }
func (s *VLANInterfaceSpec) Measurement() string { return "vlan_interface" }

func (s *VLANInterfaceSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	vlan := domain.VLANInterface{
		ID: id, Name: pairs["name"], VLANID: pairs["vlan-id"],
		Interface: pairs["interface"], MTU: pairs["mtu"],
		Running: mikrotik.ParseBool(pairs["running"]),
		Disabled: mikrotik.ParseBool(pairs["disabled"]),
		Comment: pairs["comment"], Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: vlan.ToTags(), Fields: vlan.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: vlan.ToCacheData(now),
	}, nil
}
