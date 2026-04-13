package domain

import (
	"fmt"
	"time"
)

// IPAddress represents an IP address assignment on an interface.
type IPAddress struct {
	ID        string    // RouterOS internal identifier.
	Address   string    // IP address with prefix (e.g., "192.168.1.1/24").
	Network   string    // Network address (e.g., "192.168.1.0").
	Interface string    // Interface name.
	Disabled  bool      // Whether address is disabled.
	Comment   string    // User comment.
	Timestamp time.Time // When this sample was collected.
}

func (a *IPAddress) ToTags() map[string]string {
	return map[string]string{
		"address":   a.Address,
		"interface": a.Interface,
	}
}

func (a *IPAddress) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"network":  a.Network,
		"disabled": a.Disabled,
		"comment":  a.Comment,
	}
}

func (a *IPAddress) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": a.ID, "address": a.Address, "network": a.Network,
		"interface": a.Interface, "disabled": fmt.Sprintf("%t", a.Disabled),
		"comment": a.Comment, "timestamp": ts.Format(time.RFC3339),
	}
}

// IPRoute represents an IP route entry.
type IPRoute struct {
	ID           string    // RouterOS internal identifier.
	DstAddress   string    // Destination network (e.g., "0.0.0.0/0").
	Gateway      string    // Next hop gateway.
	Distance     string    // Administrative distance.
	RoutingTable string    // Routing table name.
	PrefSrc      string    // Preferred source address.
	Scope        string    // Route scope.
	TargetScope  string    // Target scope.
	Disabled     bool      // Whether route is disabled.
	Comment      string    // User comment.
	Timestamp    time.Time // When this sample was collected.
}

func (r *IPRoute) ToTags() map[string]string {
	return map[string]string{
		"dst_address":   r.DstAddress,
		"gateway":       r.Gateway,
		"routing_table": r.RoutingTable,
	}
}

func (r *IPRoute) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"distance":  r.Distance,
		"pref_src":  r.PrefSrc,
		"scope":     r.Scope,
		"disabled":  r.Disabled,
		"comment":   r.Comment,
	}
}

func (r *IPRoute) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": r.ID, "dst_address": r.DstAddress, "gateway": r.Gateway,
		"distance": r.Distance, "routing_table": r.RoutingTable,
		"pref_src": r.PrefSrc, "scope": r.Scope, "target_scope": r.TargetScope,
		"disabled": fmt.Sprintf("%t", r.Disabled), "comment": r.Comment,
		"timestamp": ts.Format(time.RFC3339),
	}
}

// ARPEntry represents an ARP table entry.
type ARPEntry struct {
	ID         string    // RouterOS internal identifier.
	Address    string    // IP address.
	MacAddress string    // MAC address.
	Interface  string    // Interface name.
	Published  bool      // Whether ARP entry is published.
	Disabled   bool      // Whether entry is disabled.
	Comment    string    // User comment.
	Timestamp  time.Time // When this sample was collected.
}

func (a *ARPEntry) ToTags() map[string]string {
	return map[string]string{
		"address":     a.Address,
		"mac_address": a.MacAddress,
		"interface":   a.Interface,
	}
}

func (a *ARPEntry) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"published": a.Published,
		"disabled":  a.Disabled,
		"comment":   a.Comment,
	}
}

func (a *ARPEntry) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": a.ID, "address": a.Address, "mac_address": a.MacAddress,
		"interface": a.Interface, "published": fmt.Sprintf("%t", a.Published),
		"disabled": fmt.Sprintf("%t", a.Disabled), "comment": a.Comment,
		"timestamp": ts.Format(time.RFC3339),
	}
}

// IPNeighborEntry represents an IP neighbor discovered via LLDP/CDP/MNDP.
type IPNeighborEntry struct {
	ID          string    // RouterOS internal identifier.
	Interface   string    // Interface where neighbor was discovered.
	Address     string    // Neighbor's IP address.
	MacAddress  string    // Neighbor's MAC address.
	Identity    string    // Neighbor's system identity.
	Platform    string    // Neighbor's platform/hardware.
	Board       string    // Neighbor's board name.
	Version     string    // Neighbor's software version.
	InterfName  string    // Neighbor's interface name.
	Timestamp   time.Time // When this sample was collected.
}

func (n *IPNeighborEntry) ToTags() map[string]string {
	return map[string]string{
		"interface":   n.Interface,
		"identity":    n.Identity,
		"mac_address": n.MacAddress,
	}
}

func (n *IPNeighborEntry) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"address":        n.Address,
		"platform":       n.Platform,
		"board":          n.Board,
		"version":        n.Version,
		"interface_name": n.InterfName,
	}
}

func (n *IPNeighborEntry) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": n.ID, "interface": n.Interface, "address": n.Address,
		"mac_address": n.MacAddress, "identity": n.Identity, "platform": n.Platform,
		"board": n.Board, "version": n.Version, "interface_name": n.InterfName,
		"timestamp": ts.Format(time.RFC3339),
	}
}
