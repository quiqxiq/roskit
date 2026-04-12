package domain

import "time"

// PPPActiveSession represents an active PPPoE/PPTP/L2TP session.
type PPPActiveSession struct {
	ID        string    // RouterOS internal identifier for this session.
	Name      string    // Username of the active session.
	Service   string    // PPP service type (e.g., "pppoe", "pptp", "l2tp").
	CallerID  string    // Identifier of the calling station (e.g., MAC address).
	Address   string    // IP address assigned to this session.
	Uptime    string    // Session duration.
	Encoding  string    // Encryption/encoding method used.
	Timestamp time.Time // When this sample was collected.
}

// ToTags returns InfluxDB-style tags for this PPP session.
func (p *PPPActiveSession) ToTags() map[string]string {
	return map[string]string{
		"name":      p.Name,
		"service":   p.Service,
		"caller_id": p.CallerID,
	}
}

// ToFields returns InfluxDB-style fields for this PPP session.
func (p *PPPActiveSession) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"address":  p.Address,
		"uptime":   p.Uptime,
		"encoding": p.Encoding,
	}
}
