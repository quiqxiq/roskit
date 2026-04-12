package spec

import (
	"fmt"
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// InterfaceStatsSpec streams real-time interface traffic statistics.
// Uses /interface/print with stats and interval for continuous throughput monitoring.
// This is the primary spec for bandwidth monitoring across all router interfaces.
type InterfaceStatsSpec struct{}

// NewInterfaceStatsSpec creates a new InterfaceStatsSpec.
func NewInterfaceStatsSpec() *InterfaceStatsSpec {
	return &InterfaceStatsSpec{}
}

// Command returns the API command for streaming interface statistics.
// Uses =follow with =interval=1s for 1-second refresh,
// and =.proplist to limit returned attributes for router CPU efficiency.
func (s *InterfaceStatsSpec) Command() []string {
	return []string{
		"/interface/print",
		"=follow",
		"=interval=1s",
		"=.proplist=name,type,rx-byte,tx-byte,rx-packet,tx-packet,rx-drop,tx-drop,rx-error,tx-error",
	}
}

// Tag returns the unique identifier for this stream.
func (s *InterfaceStatsSpec) Tag() string {
	return "iface-stats"
}

// Measurement returns the InfluxDB measurement name.
func (s *InterfaceStatsSpec) Measurement() string {
	return "interface_stats"
}

// Parse converts a raw API sentence into an InterfaceStats TelemetryEvent.
// Handles .dead=yes for dynamically removed interfaces (e.g., VPN tunnels).
func (s *InterfaceStatsSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
	pairs := sentence.Map
	if len(pairs) == 0 {
		return nil, nil
	}

	name := pairs["name"]
	if name == "" {
		return nil, nil
	}

	now := time.Now()

	// Check for dead entity (interface removed).
	if mikrotik.IsDead(pairs) {
		return &domain.TelemetryEvent{
			RouterID:    routerID,
			Measurement: s.Measurement(),
			Type:        domain.EventDead,
			Timestamp:   now,
			CacheKey:    mikrotik.FormatCacheKey(routerID, s.Measurement(), name),
		}, nil
	}

	stats := domain.InterfaceStats{
		Name:      name,
		Type:      pairs["type"],
		RxByte:    mikrotik.ParseUint64(pairs["rx-byte"]),
		TxByte:    mikrotik.ParseUint64(pairs["tx-byte"]),
		RxPacket:  mikrotik.ParseUint64(pairs["rx-packet"]),
		TxPacket:  mikrotik.ParseUint64(pairs["tx-packet"]),
		RxDrop:    mikrotik.ParseUint64(pairs["rx-drop"]),
		TxDrop:    mikrotik.ParseUint64(pairs["tx-drop"]),
		RxError:   mikrotik.ParseUint64(pairs["rx-error"]),
		TxError:   mikrotik.ParseUint64(pairs["tx-error"]),
		Timestamp: now,
	}

	return &domain.TelemetryEvent{
		RouterID:    routerID,
		Measurement: s.Measurement(),
		Tags:        stats.ToTags(),
		Fields:      stats.ToFields(),
		Timestamp:   now,
		Type:        domain.EventUpdate,
		CacheKey:    mikrotik.FormatCacheKey(routerID, s.Measurement(), name),
		CacheData: map[string]string{
			"name":      stats.Name,
			"type":      stats.Type,
			"rx_byte":   fmt.Sprintf("%d", stats.RxByte),
			"tx_byte":   fmt.Sprintf("%d", stats.TxByte),
			"rx_packet": fmt.Sprintf("%d", stats.RxPacket),
			"tx_packet": fmt.Sprintf("%d", stats.TxPacket),
			"rx_drop":   fmt.Sprintf("%d", stats.RxDrop),
			"tx_drop":   fmt.Sprintf("%d", stats.TxDrop),
			"rx_error":  fmt.Sprintf("%d", stats.RxError),
			"tx_error":  fmt.Sprintf("%d", stats.TxError),
			"timestamp": now.Format(time.RFC3339),
		},
	}, nil
}
