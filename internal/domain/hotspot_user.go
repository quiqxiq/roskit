package domain

import (
	"fmt"
	"time"
)

// HotspotUser represents a user account configured for hotspot access.
type HotspotUser struct {
	ID        string    // RouterOS internal identifier.
	Server    string    // Hotspot server name.
	Name      string    // Username.
	Profile   string    // User profile name.
	MacAddr   string    // Bound MAC address.
	Uptime    string    // Total uptime.
	BytesIn   string    // Bytes received.
	BytesOut  string    // Bytes transmitted.
	Disabled  bool      // Whether the user is disabled.
	Comment   string    // User comment.
	Timestamp time.Time // When this sample was collected.
}

func (u *HotspotUser) ToTags() map[string]string {
	return map[string]string{
		"server":  u.Server,
		"name":    u.Name,
		"profile": u.Profile,
	}
}

func (u *HotspotUser) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"mac_address": u.MacAddr,
		"uptime":      u.Uptime,
		"bytes_in":    u.BytesIn,
		"bytes_out":   u.BytesOut,
		"disabled":    u.Disabled,
		"comment":     u.Comment,
	}
}

func (u *HotspotUser) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":          u.ID,
		"server":      u.Server,
		"name":        u.Name,
		"profile":     u.Profile,
		"mac_address": u.MacAddr,
		"uptime":      u.Uptime,
		"bytes_in":    u.BytesIn,
		"bytes_out":   u.BytesOut,
		"disabled":    fmt.Sprintf("%t", u.Disabled),
		"comment":     u.Comment,
		"timestamp":   ts.Format(time.RFC3339),
	}
}
