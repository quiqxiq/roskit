package spec

import (
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// CAPsMANRegistrationSpec streams CAPsMAN client registrations across all managed APs.
type CAPsMANRegistrationSpec struct{}

func NewCAPsMANRegistrationSpec() *CAPsMANRegistrationSpec { return &CAPsMANRegistrationSpec{} }

func (s *CAPsMANRegistrationSpec) Command() []string {
	return []string{
		"/caps-man/registration-table/print",
		"=follow",
		"=.proplist=.id,interface,mac-address,signal-strength,tx-rate,rx-rate,uptime,bytes,packets",
	}
}

func (s *CAPsMANRegistrationSpec) Tag() string         { return "capsman-reg" }
func (s *CAPsMANRegistrationSpec) Measurement() string { return "capsman_registration" }

func (s *CAPsMANRegistrationSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
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

	reg := domain.CAPsMANRegistration{
		ID: id, Interface: pairs["interface"], MacAddress: pairs["mac-address"],
		SignalStrength: pairs["signal-strength"], TxRate: pairs["tx-rate"],
		RxRate: pairs["rx-rate"], Uptime: pairs["uptime"],
		Bytes: pairs["bytes"], Packets: pairs["packets"], Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID: routerID, Measurement: s.Measurement(),
		Tags: reg.ToTags(), Fields: reg.ToFields(),
		Timestamp: now, Type: domain.EventUpdate,
		CacheKey:  mikrotik.FormatCacheKey(routerID, s.Measurement(), id),
		CacheData: reg.ToCacheData(now),
	}, nil
}
