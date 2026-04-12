package spec

import (
	"fmt"
	"time"

	"github.com/go-routeros/routeros/v3/proto"
	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// SystemResourceSpec streams system-level metrics at a fixed interval.
// Uses /system/resource/print with stats and interval=5s for periodic CPU,
// memory, and disk usage monitoring.
type SystemResourceSpec struct{}

// NewSystemResourceSpec creates a new SystemResourceSpec.
func NewSystemResourceSpec() *SystemResourceSpec {
	return &SystemResourceSpec{}
}

// Command returns the API command for streaming system resource metrics.
// /system/resource is a singleton (not a list), so =follow is not supported.
// Only =interval is used for periodic polling, and =.proplist limits returned attributes.
func (s *SystemResourceSpec) Command() []string {
	return []string{
		"/system/resource/print",
		"=interval=5s",
		"=.proplist=cpu-load,free-memory,total-memory,free-hdd-space,total-hdd-space,uptime,board-name,version",
	}
}

// Tag returns the unique identifier for this stream.
func (s *SystemResourceSpec) Tag() string {
	return "sys-resource"
}

// Measurement returns the InfluxDB measurement name.
func (s *SystemResourceSpec) Measurement() string {
	return "system_resource"
}

// Parse converts a raw API sentence into a SystemResource TelemetryEvent.
// System resources don't typically have .dead events since there's only one
// system resource per router, but we handle it for completeness.
func (s *SystemResourceSpec) Parse(routerID string, sentence *proto.Sentence) (*domain.TelemetryEvent, error) {
	pairs := sentence.Map
	if len(pairs) == 0 {
		return nil, nil
	}

	now := time.Now()

	resource := domain.SystemResource{
		CPULoad:       mikrotik.ParseUint64(pairs["cpu-load"]),
		FreeMemory:    mikrotik.ParseUint64(pairs["free-memory"]),
		TotalMemory:   mikrotik.ParseUint64(pairs["total-memory"]),
		FreeHDDSpace:  mikrotik.ParseUint64(pairs["free-hdd-space"]),
		TotalHDDSpace: mikrotik.ParseUint64(pairs["total-hdd-space"]),
		Uptime:        pairs["uptime"],
		BoardName:     pairs["board-name"],
		Version:       pairs["version"],
		Timestamp:     now,
	}

	cacheKey := mikrotik.FormatCacheKey(routerID, s.Measurement(), "system")

	return &domain.TelemetryEvent{
		RouterID:    routerID,
		Measurement: s.Measurement(),
		Tags:        resource.ToTags(),
		Fields:      resource.ToFields(),
		Timestamp:   now,
		Type:        domain.EventUpdate,
		CacheKey:    cacheKey,
		CacheData: map[string]string{
			"cpu_load":        fmt.Sprintf("%d", resource.CPULoad),
			"free_memory":     fmt.Sprintf("%d", resource.FreeMemory),
			"total_memory":    fmt.Sprintf("%d", resource.TotalMemory),
			"free_hdd_space":  fmt.Sprintf("%d", resource.FreeHDDSpace),
			"total_hdd_space": fmt.Sprintf("%d", resource.TotalHDDSpace),
			"uptime":          resource.Uptime,
			"board_name":      resource.BoardName,
			"version":         resource.Version,
			"timestamp":       now.Format(time.RFC3339),
		},
	}, nil
}
