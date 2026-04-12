package domain

import "time"

// HotspotActiveUser represents an active hotspot user session.
type HotspotActiveUser struct {
	ID         string    // RouterOS internal identifier for this session.
	User       string    // Username of the active session.
	Address    string    // IP address assigned to this user.
	MacAddress string    // MAC address of the user's device.
	Uptime     string    // Session duration.
	BytesIn    uint64    // Total bytes received by the user.
	BytesOut   uint64    // Total bytes sent by the user.
	Timestamp  time.Time // When this sample was collected.
}

// ToTags returns InfluxDB-style tags for this hotspot user.
func (u *HotspotActiveUser) ToTags() map[string]string {
	return map[string]string{
		"user":        u.User,
		"address":     u.Address,
		"mac_address": u.MacAddress,
	}
}

// ToFields returns InfluxDB-style fields for this hotspot user.
func (u *HotspotActiveUser) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"uptime":    u.Uptime,
		"bytes_in":  int64(u.BytesIn),
		"bytes_out": int64(u.BytesOut),
	}
}
