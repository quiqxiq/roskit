package domain

import (
	"fmt"
	"time"
)

// PPPProfile represents a PPP profile configuration.
// Profiles define connection parameters applied to PPP secrets/users.
type PPPProfile struct {
	ID             string    // RouterOS internal identifier.
	Name           string    // Profile name (e.g., "default", "hotspot").
	LocalAddress   string    // Local IP address assigned to the server side.
	RemoteAddress  string    // Remote IP pool or address for the client side.
	RateLimit      string    // Speed limit (e.g., "10M/10M" for rx/tx).
	AddressList    string    // Address list to add the client to.
	DNSServer      string    // DNS server assigned to the client.
	OnUp           string    // Script to run when connection comes up.
	OnDown         string    // Script to run when connection goes down.
	SessionTimeout string    // Maximum session time (e.g., "1h").
	IdleTimeout    string    // Idle timeout before disconnect.
	Comment        string    // User comment.
	Timestamp      time.Time // When this sample was collected.
}

// ToTags returns InfluxDB-style tags for this profile.
func (p *PPPProfile) ToTags() map[string]string {
	return map[string]string{
		"name": p.Name,
	}
}

// ToFields returns InfluxDB-style fields for this profile.
func (p *PPPProfile) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"local_address":   p.LocalAddress,
		"remote_address":  p.RemoteAddress,
		"rate_limit":      p.RateLimit,
		"address_list":    p.AddressList,
		"dns_server":      p.DNSServer,
		"session_timeout": p.SessionTimeout,
		"idle_timeout":    p.IdleTimeout,
	}
}

// ToCacheData returns a flat key-value map for Redis HSET.
func (p *PPPProfile) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":              p.ID,
		"name":            p.Name,
		"local_address":   p.LocalAddress,
		"remote_address":  p.RemoteAddress,
		"rate_limit":      p.RateLimit,
		"address_list":    p.AddressList,
		"dns_server":      p.DNSServer,
		"on_up":           p.OnUp,
		"on_down":         p.OnDown,
		"session_timeout": p.SessionTimeout,
		"idle_timeout":    p.IdleTimeout,
		"comment":         p.Comment,
		"timestamp":       ts.Format(time.RFC3339),
	}
}

// PPPSecret represents a PPP user account (secret).
type PPPSecret struct {
	ID            string    // RouterOS internal identifier.
	Name          string    // Username.
	Service       string    // Service type: any, pppoe, pptp, l2tp, etc.
	CallerID      string    // Allowed caller-id (MAC address filter).
	Profile       string    // Profile assigned to this secret.
	LocalAddress  string    // Local IP address override.
	RemoteAddress string    // Remote IP address override.
	Routes        string    // Static routes pushed to the client.
	LimitBytesIn  uint64    // Download byte limit (0 = unlimited).
	LimitBytesOut uint64    // Upload byte limit (0 = unlimited).
	Disabled      bool      // Whether this secret is disabled.
	Comment       string    // User comment.
	Timestamp     time.Time // When this sample was collected.
}

// ToTags returns InfluxDB-style tags for this secret.
func (s *PPPSecret) ToTags() map[string]string {
	return map[string]string{
		"name":    s.Name,
		"service": s.Service,
		"profile": s.Profile,
	}
}

// ToFields returns InfluxDB-style fields for this secret.
func (s *PPPSecret) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"caller_id":       s.CallerID,
		"local_address":   s.LocalAddress,
		"remote_address":  s.RemoteAddress,
		"routes":          s.Routes,
		"limit_bytes_in":  int64(s.LimitBytesIn),
		"limit_bytes_out": int64(s.LimitBytesOut),
		"disabled":        s.Disabled,
	}
}

// ToCacheData returns a flat key-value map for Redis HSET.
func (s *PPPSecret) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":              s.ID,
		"name":            s.Name,
		"service":         s.Service,
		"caller_id":       s.CallerID,
		"profile":         s.Profile,
		"local_address":   s.LocalAddress,
		"remote_address":  s.RemoteAddress,
		"routes":          s.Routes,
		"limit_bytes_in":  fmt.Sprintf("%d", s.LimitBytesIn),
		"limit_bytes_out": fmt.Sprintf("%d", s.LimitBytesOut),
		"disabled":        fmt.Sprintf("%t", s.Disabled),
		"comment":         s.Comment,
		"timestamp":       ts.Format(time.RFC3339),
	}
}
