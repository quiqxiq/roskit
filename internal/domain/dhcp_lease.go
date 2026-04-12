package domain

import (
	"fmt"
	"time"
)

// DHCPLease represents a DHCP server lease entry.
type DHCPLease struct {
	ID              string    // RouterOS internal identifier.
	Address         string    // IP address assigned to the client.
	MacAddress      string    // Client MAC address.
	ClientID        string    // DHCP client identifier.
	Server          string    // DHCP server name that issued this lease.
	LeaseTime       string    // Lease duration (e.g., "10m", "1d").
	Comment         string    // User comment.
	Disabled        bool      // Whether this lease is disabled.
	BlockAccess     bool      // Whether access is blocked for this client.
	RateLimit       string    // Speed limit applied to this lease.
	Routes          string    // Static routes pushed to the client.
	AddressLists    string    // Address lists the client is added to.
	DHCPOption      string    // DHCP options assigned.
	DHCPOptionSet   string    // DHCP option set assigned.
	AlwaysBroadcast bool      // Whether to always broadcast replies.
	Timestamp       time.Time // When this sample was collected.
}

// ToTags returns InfluxDB-style tags for this DHCP lease.
func (d *DHCPLease) ToTags() map[string]string {
	return map[string]string{
		"address":     d.Address,
		"mac_address": d.MacAddress,
		"server":      d.Server,
	}
}

// ToFields returns InfluxDB-style fields for this DHCP lease.
func (d *DHCPLease) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"client_id":    d.ClientID,
		"lease_time":   d.LeaseTime,
		"disabled":     d.Disabled,
		"block_access": d.BlockAccess,
		"rate_limit":   d.RateLimit,
		"comment":      d.Comment,
	}
}

// ToCacheData returns a flat key-value map for Redis HSET.
func (d *DHCPLease) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":               d.ID,
		"address":          d.Address,
		"mac_address":      d.MacAddress,
		"client_id":        d.ClientID,
		"server":           d.Server,
		"lease_time":       d.LeaseTime,
		"comment":          d.Comment,
		"disabled":         fmt.Sprintf("%t", d.Disabled),
		"block_access":     fmt.Sprintf("%t", d.BlockAccess),
		"rate_limit":       d.RateLimit,
		"routes":           d.Routes,
		"address_lists":    d.AddressLists,
		"dhcp_option":      d.DHCPOption,
		"dhcp_option_set":  d.DHCPOptionSet,
		"always_broadcast": fmt.Sprintf("%t", d.AlwaysBroadcast),
		"timestamp":        ts.Format(time.RFC3339),
	}
}
