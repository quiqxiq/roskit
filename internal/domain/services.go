package domain

import (
	"fmt"
	"time"
)

// CAPsMANInterface represents a CAPsMAN managed wireless interface (AP).
type CAPsMANInterface struct {
	ID           string    // RouterOS internal identifier.
	Name         string    // Interface name.
	RadioName    string    // Radio hardware name.
	RadioMAC     string    // Radio MAC address.
	MasterIface  string    // Master CAPsMAN interface.
	CurrentState string    // State (running, disabled).
	Bound        bool      // Whether interface is bound to a CAP.
	Inactive     bool      // Whether interface is inactive.
	Disabled     bool      // Whether interface is disabled.
	Comment      string    // User comment.
	Timestamp    time.Time // When this sample was collected.
}

func (c *CAPsMANInterface) ToTags() map[string]string {
	return map[string]string{
		"name":          c.Name,
		"radio_mac":     c.RadioMAC,
		"current_state": c.CurrentState,
	}
}

func (c *CAPsMANInterface) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"radio_name":   c.RadioName,
		"master_iface": c.MasterIface,
		"bound":        c.Bound,
		"inactive":     c.Inactive,
		"disabled":     c.Disabled,
		"comment":      c.Comment,
	}
}

func (c *CAPsMANInterface) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": c.ID, "name": c.Name, "radio_name": c.RadioName,
		"radio_mac": c.RadioMAC, "master_iface": c.MasterIface,
		"current_state": c.CurrentState, "bound": fmt.Sprintf("%t", c.Bound),
		"inactive": fmt.Sprintf("%t", c.Inactive),
		"disabled": fmt.Sprintf("%t", c.Disabled),
		"comment":  c.Comment, "timestamp": ts.Format(time.RFC3339),
	}
}

// CAPsMANRegistration represents a client registration on a CAPsMAN-managed AP.
type CAPsMANRegistration struct {
	ID             string    // RouterOS internal identifier.
	Interface      string    // CAPsMAN interface name.
	MacAddress     string    // Client MAC address.
	SignalStrength string    // Signal strength (dBm).
	TxRate         string    // Transmit rate.
	RxRate         string    // Receive rate.
	Uptime         string    // Client session uptime.
	Bytes          string    // Total bytes (rx/tx).
	Packets        string    // Total packets.
	Timestamp      time.Time // When this sample was collected.
}

func (r *CAPsMANRegistration) ToTags() map[string]string {
	return map[string]string{
		"interface":   r.Interface,
		"mac_address": r.MacAddress,
	}
}

func (r *CAPsMANRegistration) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"signal_strength": r.SignalStrength,
		"tx_rate":         r.TxRate,
		"rx_rate":         r.RxRate,
		"uptime":          r.Uptime,
		"bytes":           r.Bytes,
		"packets":         r.Packets,
	}
}

func (r *CAPsMANRegistration) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": r.ID, "interface": r.Interface, "mac_address": r.MacAddress,
		"signal_strength": r.SignalStrength, "tx_rate": r.TxRate,
		"rx_rate": r.RxRate, "uptime": r.Uptime, "bytes": r.Bytes,
		"packets": r.Packets, "timestamp": ts.Format(time.RFC3339),
	}
}

// IPPoolUsed represents a used address from an IP pool.
type IPPoolUsed struct {
	ID        string    // RouterOS internal identifier.
	Pool      string    // Pool name.
	Address   string    // Used IP address.
	Owner     string    // Owner (dhcp, ppp, etc.).
	Info      string    // Additional info (MAC, username, etc.).
	Timestamp time.Time // When this sample was collected.
}

func (u *IPPoolUsed) ToTags() map[string]string {
	return map[string]string{
		"pool":    u.Pool,
		"address": u.Address,
		"owner":   u.Owner,
	}
}

func (u *IPPoolUsed) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"info": u.Info,
	}
}

func (u *IPPoolUsed) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": u.ID, "pool": u.Pool, "address": u.Address,
		"owner": u.Owner, "info": u.Info,
		"timestamp": ts.Format(time.RFC3339),
	}
}

// DNSCacheEntry represents a DNS cache entry.
type DNSCacheEntry struct {
	ID        string    // RouterOS internal identifier.
	Name      string    // Domain name.
	Address   string    // Resolved IP address.
	TTL       string    // Time-to-live remaining.
	Type      string    // Record type (A, AAAA, CNAME, etc.).
	Timestamp time.Time // When this sample was collected.
}

func (d *DNSCacheEntry) ToTags() map[string]string {
	return map[string]string{
		"name":        d.Name,
		"record_type": d.Type,
	}
}

func (d *DNSCacheEntry) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"address": d.Address,
		"ttl":     d.TTL,
	}
}

func (d *DNSCacheEntry) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": d.ID, "name": d.Name, "address": d.Address,
		"ttl": d.TTL, "record_type": d.Type,
		"timestamp": ts.Format(time.RFC3339),
	}
}

// NetwatchEntry represents a Netwatch host monitoring entry.
type NetwatchEntry struct {
	ID        string    // RouterOS internal identifier.
	Host      string    // Target host/IP.
	Status    string    // Status (up, down, unknown).
	Interval  string    // Check interval.
	Timeout   string    // Check timeout.
	Since     string    // Time since current status.
	Comment   string    // User comment.
	Disabled  bool      // Whether entry is disabled.
	Type      string    // Check type (icmp, tcp-conn, http-get, etc.).
	Timestamp time.Time // When this sample was collected.
}

func (n *NetwatchEntry) ToTags() map[string]string {
	return map[string]string{
		"host":       n.Host,
		"status":     n.Status,
		"check_type": n.Type,
	}
}

func (n *NetwatchEntry) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"check_interval": n.Interval,
		"timeout":        n.Timeout,
		"since":          n.Since,
		"disabled":       n.Disabled,
		"comment":        n.Comment,
	}
}

func (n *NetwatchEntry) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": n.ID, "host": n.Host, "status": n.Status,
		"check_type": n.Type, "check_interval": n.Interval,
		"timeout": n.Timeout, "since": n.Since,
		"disabled": fmt.Sprintf("%t", n.Disabled),
		"comment":  n.Comment, "timestamp": ts.Format(time.RFC3339),
	}
}

// UserActive represents an active user session on the router.
type UserActive struct {
	ID        string    // RouterOS internal identifier.
	Name      string    // Username.
	Address   string    // Source IP address.
	Via       string    // Access method (api, winbox, ssh, telnet).
	When      string    // Login time.
	Group     string    // User group.
	Timestamp time.Time // When this sample was collected.
}

func (u *UserActive) ToTags() map[string]string {
	return map[string]string{
		"name":    u.Name,
		"address": u.Address,
		"via":     u.Via,
	}
}

func (u *UserActive) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"when":  u.When,
		"group": u.Group,
	}
}

func (u *UserActive) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": u.ID, "name": u.Name, "address": u.Address,
		"via": u.Via, "when": u.When, "group": u.Group,
		"timestamp": ts.Format(time.RFC3339),
	}
}
