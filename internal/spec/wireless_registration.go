package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// WirelessRegistrationSpec streams wireless client registration table changes.
// Tracks WiFi client signal strength, rates, and connection status.
type WirelessRegistrationSpec struct{}

func NewWirelessRegistrationSpec() *WirelessRegistrationSpec { return &WirelessRegistrationSpec{} }

func (s *WirelessRegistrationSpec) Command() []string {
	return []string{
		"/interface/wireless/registration-table/print",
		"=follow",
		"=.proplist=.id,interface,mac-address,signal-strength,tx-rate,rx-rate,uptime,last-activity,bytes,packets",
	}
}

func (s *WirelessRegistrationSpec) Tag() string         { return "wireless-reg" }
func (s *WirelessRegistrationSpec) Measurement() string { return "wireless_registration" }

func (s *WirelessRegistrationSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	reg := domain.WirelessRegistration{
		ID: id, Interface: pairs["interface"], MacAddress: pairs["mac-address"],
		SignalStrength: pairs["signal-strength"], TxRate: pairs["tx-rate"],
		RxRate: pairs["rx-rate"], Uptime: pairs["uptime"],
		LastActivity: pairs["last-activity"], Bytes: pairs["bytes"],
		Packets: pairs["packets"], Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: reg.ToTags(), Fields: reg.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: reg.ToCacheData(now),
	}, nil
}
