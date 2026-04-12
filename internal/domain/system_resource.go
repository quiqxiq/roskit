package domain

import "time"

// SystemResource represents system-level metrics of a MikroTik router.
type SystemResource struct {
	CPULoad       uint64    // Current CPU utilization percentage.
	FreeMemory    uint64    // Available memory in bytes.
	TotalMemory   uint64    // Total installed memory in bytes.
	FreeHDDSpace  uint64    // Available disk space in bytes.
	TotalHDDSpace uint64    // Total disk space in bytes.
	Uptime        string    // System uptime as reported by RouterOS.
	BoardName     string    // Hardware model name.
	Version       string    // RouterOS version string.
	Timestamp     time.Time // When this sample was collected.
}

// ToTags returns InfluxDB-style tags for this system resource snapshot.
func (r *SystemResource) ToTags() map[string]string {
	return map[string]string{
		"board_name": r.BoardName,
		"version":    r.Version,
	}
}

// ToFields returns InfluxDB-style fields for this system resource snapshot.
func (r *SystemResource) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"cpu_load":        int64(r.CPULoad),
		"free_memory":     int64(r.FreeMemory),
		"total_memory":    int64(r.TotalMemory),
		"free_hdd_space":  int64(r.FreeHDDSpace),
		"total_hdd_space": int64(r.TotalHDDSpace),
		"uptime":          r.Uptime,
	}
}
