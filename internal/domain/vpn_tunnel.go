package domain

import (
	"fmt"
	"time"
)

// IPsecActivePeer represents an active IPsec peer connection.
type IPsecActivePeer struct {
	ID            string    // RouterOS internal identifier.
	RemoteAddress string    // Remote peer IP address.
	LocalAddress  string    // Local IP address.
	State         string    // Peer state (established, connecting).
	Side          string    // Initiator or responder.
	Uptime        string    // Session uptime.
	RxBytes       string    // Received bytes.
	TxBytes       string    // Transmitted bytes.
	RxPackets     string    // Received packets.
	TxPackets     string    // Transmitted packets.
	DynAddr       string    // Dynamic address (if mode-config).
	Responder     bool      // Whether peer is responder.
	Timestamp     time.Time // When this sample was collected.
}

func (p *IPsecActivePeer) ToTags() map[string]string {
	return map[string]string{
		"remote_address": p.RemoteAddress,
		"local_address":  p.LocalAddress,
		"state":          p.State,
	}
}

func (p *IPsecActivePeer) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"side":       p.Side,
		"uptime":     p.Uptime,
		"rx_bytes":   p.RxBytes,
		"tx_bytes":   p.TxBytes,
		"rx_packets": p.RxPackets,
		"tx_packets": p.TxPackets,
		"responder":  p.Responder,
	}
}

func (p *IPsecActivePeer) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": p.ID, "remote_address": p.RemoteAddress, "local_address": p.LocalAddress,
		"state": p.State, "side": p.Side, "uptime": p.Uptime,
		"rx_bytes": p.RxBytes, "tx_bytes": p.TxBytes,
		"rx_packets": p.RxPackets, "tx_packets": p.TxPackets,
		"responder": fmt.Sprintf("%t", p.Responder), "timestamp": ts.Format(time.RFC3339),
	}
}

// IPsecPolicy represents an IPsec security policy.
type IPsecPolicy struct {
	ID        string    // RouterOS internal identifier.
	Peer      string    // Associated peer name.
	SrcAddr   string    // Source address.
	DstAddr   string    // Destination address.
	Protocol  string    // Protocol filter (all, tcp, udp).
	Action    string    // Action (encrypt, discard, none).
	Level     string    // IPsec level (require, unique, use).
	Disabled  bool      // Whether policy is disabled.
	Comment   string    // User comment.
	Timestamp time.Time // When this sample was collected.
}

func (p *IPsecPolicy) ToTags() map[string]string {
	return map[string]string{
		"peer":        p.Peer,
		"src_address": p.SrcAddr,
		"dst_address": p.DstAddr,
		"action":      p.Action,
	}
}

func (p *IPsecPolicy) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"protocol": p.Protocol,
		"level":    p.Level,
		"disabled": p.Disabled,
		"comment":  p.Comment,
	}
}

func (p *IPsecPolicy) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": p.ID, "peer": p.Peer, "src_address": p.SrcAddr,
		"dst_address": p.DstAddr, "protocol": p.Protocol, "action": p.Action,
		"level": p.Level, "disabled": fmt.Sprintf("%t", p.Disabled),
		"comment": p.Comment, "timestamp": ts.Format(time.RFC3339),
	}
}

// WireguardPeer represents a WireGuard peer entry.
type WireguardPeer struct {
	ID             string    // RouterOS internal identifier.
	Interface      string    // WireGuard interface name.
	PublicKey      string    // Peer public key.
	EndpointAddr   string    // Endpoint address.
	EndpointPort   string    // Endpoint port.
	AllowedAddress string    // Allowed addresses.
	LastHandshake  string    // Time since last handshake.
	Rx             string    // Received bytes.
	Tx             string    // Transmitted bytes.
	Disabled       bool      // Whether peer is disabled.
	Comment        string    // User comment.
	Timestamp      time.Time // When this sample was collected.
}

func (w *WireguardPeer) ToTags() map[string]string {
	return map[string]string{
		"interface":     w.Interface,
		"endpoint_addr": w.EndpointAddr,
	}
}

func (w *WireguardPeer) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"public_key":      w.PublicKey,
		"endpoint_port":   w.EndpointPort,
		"allowed_address": w.AllowedAddress,
		"last_handshake":  w.LastHandshake,
		"rx":              w.Rx,
		"tx":              w.Tx,
		"disabled":        w.Disabled,
		"comment":         w.Comment,
	}
}

func (w *WireguardPeer) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": w.ID, "interface": w.Interface, "public_key": w.PublicKey,
		"endpoint_addr": w.EndpointAddr, "endpoint_port": w.EndpointPort,
		"allowed_address": w.AllowedAddress, "last_handshake": w.LastHandshake,
		"rx": w.Rx, "tx": w.Tx, "disabled": fmt.Sprintf("%t", w.Disabled),
		"comment": w.Comment, "timestamp": ts.Format(time.RFC3339),
	}
}

// TunnelInterface represents a generic tunnel interface (EoIP, GRE, L2TP).
type TunnelInterface struct {
	ID            string    // RouterOS internal identifier.
	Name          string    // Interface name.
	Type          string    // Tunnel type (eoip, gre, l2tp).
	RemoteAddress string    // Remote endpoint address.
	LocalAddress  string    // Local endpoint address.
	Running       bool      // Whether tunnel is running.
	Disabled      bool      // Whether tunnel is disabled.
	MTU           string    // Maximum transmission unit.
	Comment       string    // User comment.
	Timestamp     time.Time // When this sample was collected.
}

func (t *TunnelInterface) ToTags() map[string]string {
	return map[string]string{
		"name":           t.Name,
		"tunnel_type":    t.Type,
		"remote_address": t.RemoteAddress,
	}
}

func (t *TunnelInterface) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"local_address": t.LocalAddress,
		"running":       t.Running,
		"disabled":      t.Disabled,
		"mtu":           t.MTU,
		"comment":       t.Comment,
	}
}

func (t *TunnelInterface) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": t.ID, "name": t.Name, "tunnel_type": t.Type,
		"remote_address": t.RemoteAddress, "local_address": t.LocalAddress,
		"running": fmt.Sprintf("%t", t.Running), "disabled": fmt.Sprintf("%t", t.Disabled),
		"mtu": t.MTU, "comment": t.Comment, "timestamp": ts.Format(time.RFC3339),
	}
}
