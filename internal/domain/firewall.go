package domain

import (
	"fmt"
	"time"
)

// FirewallRule represents a firewall filter/NAT/mangle rule with hit counters.
type FirewallRule struct {
	ID         string    // RouterOS internal identifier.
	Chain      string    // Chain name (input, forward, output, srcnat, dstnat).
	Action     string    // Action (accept, drop, reject, masquerade, etc.).
	Comment    string    // User comment (used as rule identifier).
	Bytes      uint64    // Total bytes matched by this rule.
	Packets    uint64    // Total packets matched by this rule.
	Disabled   bool      // Whether rule is disabled.
	SrcAddress string    // Source address filter.
	DstAddress string    // Destination address filter.
	Protocol   string    // Protocol filter (tcp, udp, icmp).
	DstPort    string    // Destination port filter.
	SrcPort    string    // Source port filter.
	InIface    string    // Input interface filter.
	OutIface   string    // Output interface filter.
	ConnState  string    // Connection state filter.
	Timestamp  time.Time // When this sample was collected.
}

func (f *FirewallRule) ToTags() map[string]string {
	return map[string]string{
		"chain":   f.Chain,
		"action":  f.Action,
		"comment": f.Comment,
	}
}

func (f *FirewallRule) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"bytes":       int64(f.Bytes),
		"packets":     int64(f.Packets),
		"disabled":    f.Disabled,
		"src_address": f.SrcAddress,
		"dst_address": f.DstAddress,
		"protocol":    f.Protocol,
		"dst_port":    f.DstPort,
	}
}

func (f *FirewallRule) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": f.ID, "chain": f.Chain, "action": f.Action, "comment": f.Comment,
		"bytes": fmt.Sprintf("%d", f.Bytes), "packets": fmt.Sprintf("%d", f.Packets),
		"disabled": fmt.Sprintf("%t", f.Disabled), "src_address": f.SrcAddress,
		"dst_address": f.DstAddress, "protocol": f.Protocol, "dst_port": f.DstPort,
		"src_port": f.SrcPort, "in_interface": f.InIface, "out_interface": f.OutIface,
		"connection_state": f.ConnState, "timestamp": ts.Format(time.RFC3339),
	}
}

// FirewallConnection represents an active connection in the connection tracker.
type FirewallConnection struct {
	ID         string    // RouterOS internal identifier.
	Protocol   string    // Protocol (tcp, udp, icmp).
	SrcAddress string    // Source address:port.
	DstAddress string    // Destination address:port.
	ReplySrc   string    // Reply source address.
	ReplyDst   string    // Reply destination address.
	TCPState   string    // TCP state (established, time-wait, etc.).
	Timeout    string    // Connection timeout remaining.
	Assured    bool      // Whether connection is assured.
	Timestamp  time.Time // When this sample was collected.
}

func (c *FirewallConnection) ToTags() map[string]string {
	return map[string]string{
		"protocol":    c.Protocol,
		"src_address": c.SrcAddress,
		"dst_address": c.DstAddress,
	}
}

func (c *FirewallConnection) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"reply_src": c.ReplySrc,
		"reply_dst": c.ReplyDst,
		"tcp_state": c.TCPState,
		"timeout":   c.Timeout,
		"assured":   c.Assured,
	}
}

func (c *FirewallConnection) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": c.ID, "protocol": c.Protocol, "src_address": c.SrcAddress,
		"dst_address": c.DstAddress, "reply_src": c.ReplySrc, "reply_dst": c.ReplyDst,
		"tcp_state": c.TCPState, "timeout": c.Timeout,
		"assured": fmt.Sprintf("%t", c.Assured), "timestamp": ts.Format(time.RFC3339),
	}
}
