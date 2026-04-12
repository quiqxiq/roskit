// Package domain defines core business entities for the roskit telemetry library.
// These entities are completely decoupled from any external libraries or infrastructure.
package domain

import "time"

// RouterConfig holds the connection parameters for a MikroTik device.
type RouterConfig struct {
	// ID is a unique identifier for this router (e.g., "core-01").
	ID string
	// Address is the MikroTik API endpoint (e.g., "192.168.88.1:8728" or "192.168.88.1:8729" for TLS).
	Address string
	// Username for API authentication.
	Username string
	// Password for API authentication.
	Password string
	// UseTLS enables TLS connection to the API port (default 8729).
	UseTLS bool

	// ReconnectInterval is how long to wait before reconnecting after a connection drop.
	// The pool uses exponential backoff starting from this value, capped at MaxReconnectInterval.
	// Default: 5s if zero.
	ReconnectInterval time.Duration
	// MaxReconnectInterval is the maximum duration between reconnection attempts.
	// Default: 60s if zero.
	MaxReconnectInterval time.Duration
	// HealthCheckInterval is how often the pool verifies connection liveness
	// by sending /system/identity/print to the router.
	// Default: 30s if zero.
	HealthCheckInterval time.Duration
	// DialTimeout is the maximum time to wait for a connection to be established.
	// Default: 10s if zero.
	DialTimeout time.Duration
}

// Defaults applies default values to zero-valued config fields.
func (c *RouterConfig) Defaults() {
	if c.ReconnectInterval <= 0 {
		c.ReconnectInterval = 5 * time.Second
	}
	if c.MaxReconnectInterval <= 0 {
		c.MaxReconnectInterval = 60 * time.Second
	}
	if c.HealthCheckInterval <= 0 {
		c.HealthCheckInterval = 30 * time.Second
	}
	if c.DialTimeout <= 0 {
		c.DialTimeout = 10 * time.Second
	}
}

// InterfaceStats represents real-time traffic statistics for a network interface.
type InterfaceStats struct {
	// Name is the interface name (e.g., "ether1", "wlan1").
	Name string
	// Type is the interface type (e.g., "ether", "wlan", "bridge").
	Type string
	// RxByte is the total received bytes counter.
	RxByte uint64
	// TxByte is the total transmitted bytes counter.
	TxByte uint64
	// RxPacket is the total received packets counter.
	RxPacket uint64
	// TxPacket is the total transmitted packets counter.
	TxPacket uint64
	// RxDrop is the total received dropped packets counter.
	RxDrop uint64
	// TxDrop is the total transmitted dropped packets counter.
	TxDrop uint64
	// RxError is the total received error packets counter.
	RxError uint64
	// TxError is the total transmitted error packets counter.
	TxError uint64
	// Timestamp is when this sample was collected.
	Timestamp time.Time
}

// ToTags returns InfluxDB-style tags for this interface.
func (s *InterfaceStats) ToTags() map[string]string {
	return map[string]string{
		"name": s.Name,
		"type": s.Type,
	}
}

// ToFields returns InfluxDB-style fields for this interface.
func (s *InterfaceStats) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"rx_byte":   int64(s.RxByte),
		"tx_byte":   int64(s.TxByte),
		"rx_packet": int64(s.RxPacket),
		"tx_packet": int64(s.TxPacket),
		"rx_drop":   int64(s.RxDrop),
		"tx_drop":   int64(s.TxDrop),
		"rx_error":  int64(s.RxError),
		"tx_error":  int64(s.TxError),
	}
}

// HotspotActiveUser represents an active hotspot user session.
type HotspotActiveUser struct {
	// ID is the RouterOS internal identifier for this session.
	ID string
	// User is the username of the active session.
	User string
	// Address is the IP address assigned to this user.
	Address string
	// MacAddress is the MAC address of the user's device.
	MacAddress string
	// Uptime is the session duration.
	Uptime string
	// BytesIn is the total bytes received by the user.
	BytesIn uint64
	// BytesOut is the total bytes sent by the user.
	BytesOut uint64
	// Timestamp is when this sample was collected.
	Timestamp time.Time
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

// PPPActiveSession represents an active PPPoE/PPTP/L2TP session.
type PPPActiveSession struct {
	// ID is the RouterOS internal identifier for this session.
	ID string
	// Name is the username of the active session.
	Name string
	// Service is the PPP service type (e.g., "pppoe", "pptp", "l2tp").
	Service string
	// CallerID is the identifier of the calling station (e.g., MAC address).
	CallerID string
	// Address is the IP address assigned to this session.
	Address string
	// Uptime is the session duration.
	Uptime string
	// Encoding is the encryption/encoding method used.
	Encoding string
	// Timestamp is when this sample was collected.
	Timestamp time.Time
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

// SystemResource represents system-level metrics of a MikroTik router.
type SystemResource struct {
	// CPULoad is the current CPU utilization percentage.
	CPULoad uint64
	// FreeMemory is the available memory in bytes.
	FreeMemory uint64
	// TotalMemory is the total installed memory in bytes.
	TotalMemory uint64
	// FreeHDDSpace is the available disk space in bytes.
	FreeHDDSpace uint64
	// TotalHDDSpace is the total disk space in bytes.
	TotalHDDSpace uint64
	// Uptime is the system uptime as reported by RouterOS.
	Uptime string
	// BoardName is the hardware model name.
	BoardName string
	// Version is the RouterOS version string.
	Version string
	// Timestamp is when this sample was collected.
	Timestamp time.Time
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
