package domain

import (
	"fmt"
	"time"
)

// BridgeHost represents a MAC address in the bridge forwarding table.
type BridgeHost struct {
	ID        string    // RouterOS internal identifier.
	Bridge    string    // Bridge interface name.
	Interface string    // Port interface name.
	MacAddr   string    // MAC address.
	VID       string    // VLAN ID (if applicable).
	OnLocal   bool      // Whether entry is local.
	Disabled  bool      // Whether entry is disabled.
	Timestamp time.Time // When this sample was collected.
}

func (b *BridgeHost) ToTags() map[string]string {
	return map[string]string{
		"bridge":      b.Bridge,
		"interface":   b.Interface,
		"mac_address": b.MacAddr,
	}
}

func (b *BridgeHost) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"vid":      b.VID,
		"on_local": b.OnLocal,
		"disabled": b.Disabled,
	}
}

func (b *BridgeHost) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": b.ID, "bridge": b.Bridge, "interface": b.Interface,
		"mac_address": b.MacAddr, "vid": b.VID,
		"on_local":  fmt.Sprintf("%t", b.OnLocal),
		"disabled":  fmt.Sprintf("%t", b.Disabled),
		"timestamp": ts.Format(time.RFC3339),
	}
}

// BridgePort represents a port in a bridge configuration.
type BridgePort struct {
	ID        string    // RouterOS internal identifier.
	Bridge    string    // Parent bridge name.
	Interface string    // Port interface name.
	Role      string    // STP role (root-port, designated-port, etc.).
	Status    string    // STP status (in-bridge, disabled).
	PortNum   string    // STP port number.
	Priority  string    // STP port priority.
	Edge      string    // Edge port setting (auto, yes, no).
	Learning  bool      // Whether port is learning MACs.
	Disabled  bool      // Whether port is admin disabled.
	Comment   string    // User comment.
	Timestamp time.Time // When this sample was collected.
}

func (p *BridgePort) ToTags() map[string]string {
	return map[string]string{
		"bridge":    p.Bridge,
		"interface": p.Interface,
		"role":      p.Role,
		"status":    p.Status,
	}
}

func (p *BridgePort) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"port_number": p.PortNum,
		"priority":    p.Priority,
		"edge":        p.Edge,
		"learning":    p.Learning,
		"disabled":    p.Disabled,
		"comment":     p.Comment,
	}
}

func (p *BridgePort) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": p.ID, "bridge": p.Bridge, "interface": p.Interface,
		"role": p.Role, "status": p.Status, "port_number": p.PortNum,
		"priority": p.Priority, "edge": p.Edge,
		"learning": fmt.Sprintf("%t", p.Learning),
		"disabled": fmt.Sprintf("%t", p.Disabled),
		"comment":  p.Comment, "timestamp": ts.Format(time.RFC3339),
	}
}

// VLANInterface represents a VLAN sub-interface.
type VLANInterface struct {
	ID        string    // RouterOS internal identifier.
	Name      string    // VLAN interface name.
	VLANID    string    // VLAN ID.
	Interface string    // Parent interface.
	MTU       string    // MTU value.
	Running   bool      // Whether interface is running.
	Disabled  bool      // Whether interface is disabled.
	Comment   string    // User comment.
	Timestamp time.Time // When this sample was collected.
}

func (v *VLANInterface) ToTags() map[string]string {
	return map[string]string{
		"name":      v.Name,
		"vlan_id":   v.VLANID,
		"interface": v.Interface,
	}
}

func (v *VLANInterface) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"mtu":      v.MTU,
		"running":  v.Running,
		"disabled": v.Disabled,
		"comment":  v.Comment,
	}
}

func (v *VLANInterface) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": v.ID, "name": v.Name, "vlan_id": v.VLANID,
		"interface": v.Interface, "mtu": v.MTU,
		"running":  fmt.Sprintf("%t", v.Running),
		"disabled": fmt.Sprintf("%t", v.Disabled),
		"comment":  v.Comment, "timestamp": ts.Format(time.RFC3339),
	}
}
