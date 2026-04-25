package model

import (
	"fmt"
	"time"
)

type InterfaceStats struct {
	Name               string
	RXBitsPerSecond    uint64
	TXBitsPerSecond    uint64
	RXPacketsPerSecond uint64
	TXPacketsPerSecond uint64
	RXErrors           uint64
	TXErrors           uint64
	RXDrops            uint64
	TXDrops            uint64
	LinkDowns          uint64
	Timestamp          time.Time
}

func (s *InterfaceStats) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"name":                  s.Name,
		"rx_bits_per_second":    fmt.Sprintf("%d", s.RXBitsPerSecond),
		"tx_bits_per_second":    fmt.Sprintf("%d", s.TXBitsPerSecond),
		"rx_packets_per_second": fmt.Sprintf("%d", s.RXPacketsPerSecond),
		"tx_packets_per_second": fmt.Sprintf("%d", s.TXPacketsPerSecond),
		"rx_errors":             fmt.Sprintf("%d", s.RXErrors),
		"tx_errors":             fmt.Sprintf("%d", s.TXErrors),
		"rx_drops":              fmt.Sprintf("%d", s.RXDrops),
		"tx_drops":              fmt.Sprintf("%d", s.TXDrops),
		"link_downs":            fmt.Sprintf("%d", s.LinkDowns),
		"timestamp":             ts.Format(time.RFC3339),
	}
}

func NewInterfaceStatsFromReply(data map[string]string) *InterfaceStats {
	now := time.Now()
	return &InterfaceStats{
		Name:               data["name"],
		RXBitsPerSecond:    pUint64(data["rx-bits-per-second"]),
		TXBitsPerSecond:    pUint64(data["tx-bits-per-second"]),
		RXPacketsPerSecond: pUint64(data["rx-packets-per-second"]),
		TXPacketsPerSecond: pUint64(data["tx-packets-per-second"]),
		RXErrors:           pUint64(data["rx-errors"]),
		TXErrors:           pUint64(data["tx-errors"]),
		RXDrops:            pUint64(data["rx-drops"]),
		TXDrops:            pUint64(data["tx-drops"]),
		LinkDowns:          pUint64(data["link-downs"]),
		Timestamp:          now,
	}
}

type Interface struct {
	ID        string
	Name      string
	Type      string
	MTU       string
	Running   bool
	Disabled  bool
	Comment   string
	RXByte    uint64
	TXByte    uint64
	RXPacket  uint64
	TXPacket  uint64
	LinkDowns uint64
	Timestamp time.Time
}

func NewInterfaceFromReply(data map[string]string) *Interface {
	now := time.Now()
	return &Interface{
		ID:        data[".id"],
		Name:      data["name"],
		Type:      data["type"],
		MTU:       data["mtu"],
		Running:   pBool(data["running"]),
		Disabled:  pBool(data["disabled"]),
		Comment:   data["comment"],
		RXByte:    pUint64(data["rx-byte"]),
		TXByte:    pUint64(data["tx-byte"]),
		RXPacket:  pUint64(data["rx-packet"]),
		TXPacket:  pUint64(data["tx-packet"]),
		LinkDowns: pUint64(data["link-downs"]),
		Timestamp: now,
	}
}

func (i *Interface) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":         i.ID,
		"name":       i.Name,
		"type":       i.Type,
		"mtu":        i.MTU,
		"running":    fmt.Sprintf("%t", i.Running),
		"disabled":   fmt.Sprintf("%t", i.Disabled),
		"comment":    i.Comment,
		"rx_byte":    fmt.Sprintf("%d", i.RXByte),
		"tx_byte":    fmt.Sprintf("%d", i.TXByte),
		"rx_packet":  fmt.Sprintf("%d", i.RXPacket),
		"tx_packet":  fmt.Sprintf("%d", i.TXPacket),
		"link_downs": fmt.Sprintf("%d", i.LinkDowns),
		"timestamp":  ts.Format(time.RFC3339),
	}
}

type DHCPLease struct {
	ID              string
	Address         string
	MACAddress      string
	ClientID        string
	Server          string
	LeaseTime       string
	Comment         string
	Disabled        bool
	BlockAccess     bool
	RateLimit       string
	Dynamic         bool
	Hostname        string
	Status          string
	Timestamp       time.Time
}

func NewDHCPLeaseFromReply(data map[string]string) *DHCPLease {
	now := time.Now()
	return &DHCPLease{
		ID:         data[".id"],
		Address:    data["address"],
		MACAddress: data["mac-address"],
		ClientID:   data["client-id"],
		Server:     data["server"],
		LeaseTime:  data["lease-time"],
		Comment:    data["comment"],
		Disabled:   pBool(data["disabled"]),
		BlockAccess: pBool(data["block-access"]),
		RateLimit:  data["rate-limit"],
		Dynamic:    pBool(data["dynamic"]),
		Hostname:   data["host-name"],
		Status:     data["status"],
		Timestamp:  now,
	}
}

func (d *DHCPLease) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":           d.ID,
		"address":      d.Address,
		"mac_address":  d.MACAddress,
		"client_id":    d.ClientID,
		"server":       d.Server,
		"lease_time":   d.LeaseTime,
		"comment":      d.Comment,
		"disabled":     fmt.Sprintf("%t", d.Disabled),
		"block_access": fmt.Sprintf("%t", d.BlockAccess),
		"rate_limit":   d.RateLimit,
		"dynamic":      fmt.Sprintf("%t", d.Dynamic),
		"host_name":    d.Hostname,
		"status":       d.Status,
		"timestamp":    ts.Format(time.RFC3339),
	}
}

func (d *DHCPLease) ToTags() map[string]string {
	return map[string]string{
		"address":     d.Address,
		"mac_address": d.MACAddress,
		"server":      d.Server,
	}
}

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

type PPPSecret struct {
	ID            string
	Name          string
	Password      string
	Profile       string
	Service       string
	CallerID      string
	LocalAddress  string
	RemoteAddress string
	Disabled      bool
	Comment       string
	Timestamp     time.Time
}

func NewPPPSecretFromReply(data map[string]string) *PPPSecret {
	now := time.Now()
	return &PPPSecret{
		ID:            data[".id"],
		Name:          data["name"],
		Password:      data["password"],
		Profile:       data["profile"],
		Service:       data["service"],
		CallerID:      data["caller-id"],
		LocalAddress:  data["local-address"],
		RemoteAddress: data["remote-address"],
		Disabled:      pBool(data["disabled"]),
		Comment:       data["comment"],
		Timestamp:     now,
	}
}

func (s *PPPSecret) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":             s.ID,
		"name":           s.Name,
		"password":       s.Password,
		"profile":        s.Profile,
		"service":        s.Service,
		"caller_id":      s.CallerID,
		"local_address":  s.LocalAddress,
		"remote_address": s.RemoteAddress,
		"disabled":       fmt.Sprintf("%t", s.Disabled),
		"comment":        s.Comment,
		"timestamp":      ts.Format(time.RFC3339),
	}
}

func (s *PPPSecret) ToTags() map[string]string {
	return map[string]string{
		"name":    s.Name,
		"profile": s.Profile,
		"service": s.Service,
	}
}

func (s *PPPSecret) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"disabled":       s.Disabled,
		"caller_id":      s.CallerID,
		"local_address":  s.LocalAddress,
		"remote_address": s.RemoteAddress,
		"comment":        s.Comment,
	}
}

type PPPActive struct {
	ID            string
	Name          string
	Service       string
	CallerID      string
	Address       string
	Uptime        string
	Encoding      string
	SessionID     string
	LimitBytesIn  uint64
	LimitBytesOut uint64
	Timestamp     time.Time
}

func NewPPPActiveFromReply(data map[string]string) *PPPActive {
	now := time.Now()
	return &PPPActive{
		ID:            data[".id"],
		Name:          data["name"],
		Service:       data["service"],
		CallerID:      data["caller-id"],
		Address:       data["address"],
		Uptime:        data["uptime"],
		Encoding:      data["encoding"],
		SessionID:     data["session-id"],
		LimitBytesIn:  pUint64(data["limit-bytes-in"]),
		LimitBytesOut: pUint64(data["limit-bytes-out"]),
		Timestamp:     now,
	}
}

func (a *PPPActive) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":              a.ID,
		"name":            a.Name,
		"service":         a.Service,
		"caller_id":       a.CallerID,
		"address":         a.Address,
		"uptime":          a.Uptime,
		"encoding":        a.Encoding,
		"session_id":      a.SessionID,
		"limit_bytes_in":  fmt.Sprintf("%d", a.LimitBytesIn),
		"limit_bytes_out": fmt.Sprintf("%d", a.LimitBytesOut),
		"timestamp":       ts.Format(time.RFC3339),
	}
}

func (a *PPPActive) ToTags() map[string]string {
	return map[string]string{
		"name":    a.Name,
		"service": a.Service,
	}
}

func (a *PPPActive) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"caller_id":       a.CallerID,
		"address":         a.Address,
		"uptime":          a.Uptime,
		"encoding":        a.Encoding,
		"session_id":      a.SessionID,
		"limit_bytes_in":  a.LimitBytesIn,
		"limit_bytes_out": a.LimitBytesOut,
	}
}

type ARPEntry struct {
	ID        string
	Address   string
	MACAddress string
	Interface string
	Status    string
	Complete  bool
	Disabled  bool
	Dynamic   bool
	Comment   string
	Timestamp time.Time
}

func NewARPEntryFromReply(data map[string]string) *ARPEntry {
	now := time.Now()
	return &ARPEntry{
		ID:         data[".id"],
		Address:    data["address"],
		MACAddress: data["mac-address"],
		Interface:  data["interface"],
		Status:     data["status"],
		Complete:   pBool(data["complete"]),
		Disabled:   pBool(data["disabled"]),
		Dynamic:    pBool(data["dynamic"]),
		Comment:    data["comment"],
		Timestamp:  now,
	}
}

func (a *ARPEntry) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":         a.ID,
		"address":    a.Address,
		"mac_address": a.MACAddress,
		"interface":  a.Interface,
		"status":     a.Status,
		"complete":   fmt.Sprintf("%t", a.Complete),
		"disabled":   fmt.Sprintf("%t", a.Disabled),
		"dynamic":    fmt.Sprintf("%t", a.Dynamic),
		"comment":    a.Comment,
		"timestamp":  ts.Format(time.RFC3339),
	}
}

type IPPool struct {
	ID        string
	Name      string
	Ranges    string
	NextPool  string
	Timestamp time.Time
}

func NewIPPoolFromReply(data map[string]string) *IPPool {
	now := time.Now()
	return &IPPool{
		ID:        data[".id"],
		Name:      data["name"],
		Ranges:    data["ranges"],
		NextPool:  data["next-pool"],
		Timestamp: now,
	}
}

func (p *IPPool) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":        p.ID,
		"name":      p.Name,
		"ranges":    p.Ranges,
		"next_pool": p.NextPool,
		"timestamp": ts.Format(time.RFC3339),
	}
}

type FirewallNAT struct {
	ID          string
	Chain       string
	Action      string
	Protocol    string
	DstAddress  string
	DstPort     string
	SrcAddress  string
	SrcPort     string
	ToAddresses string
	ToPorts     string
	Comment     string
	Disabled    bool
	InInterface string
	OutInterface string
	Log         bool
	LogPrefix   string
	Timestamp   time.Time
}

func NewFirewallNATFromReply(data map[string]string) *FirewallNAT {
	now := time.Now()
	return &FirewallNAT{
		ID:           data[".id"],
		Chain:        data["chain"],
		Action:       data["action"],
		Protocol:     data["protocol"],
		DstAddress:   data["dst-address"],
		DstPort:      data["dst-port"],
		SrcAddress:   data["src-address"],
		SrcPort:      data["src-port"],
		ToAddresses:  data["to-addresses"],
		ToPorts:      data["to-ports"],
		Comment:      data["comment"],
		Disabled:     pBool(data["disabled"]),
		InInterface:  data["in-interface"],
		OutInterface: data["out-interface"],
		Log:          pBool(data["log"]),
		LogPrefix:    data["log-prefix"],
		Timestamp:    now,
	}
}

func (n *FirewallNAT) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":           n.ID,
		"chain":        n.Chain,
		"action":       n.Action,
		"protocol":     n.Protocol,
		"dst_address":  n.DstAddress,
		"dst_port":     n.DstPort,
		"src_address":  n.SrcAddress,
		"src_port":     n.SrcPort,
		"to_addresses": n.ToAddresses,
		"to_ports":     n.ToPorts,
		"comment":      n.Comment,
		"disabled":     fmt.Sprintf("%t", n.Disabled),
		"in_interface":  n.InInterface,
		"out_interface": n.OutInterface,
		"log":          fmt.Sprintf("%t", n.Log),
		"log_prefix":   n.LogPrefix,
		"timestamp":    ts.Format(time.RFC3339),
	}
}

type QueueSimple struct {
	ID             string
	Name           string
	Target         string
	Parent         string
	PacketMark     string
	RateLimit      string
	MaxLimit       string
	BurstLimit     string
	BurstThreshold string
	BurstTime      string
	Priority       string
	Queue          string
	Comment        string
	Disabled       bool
	Dynamic        bool
	Timestamp      time.Time
}

func NewQueueSimpleFromReply(data map[string]string) *QueueSimple {
	now := time.Now()
	return &QueueSimple{
		ID:             data[".id"],
		Name:           data["name"],
		Target:         data["target"],
		Parent:         data["parent"],
		PacketMark:     data["packet-mark"],
		RateLimit:      data["rate-limit"],
		MaxLimit:       data["max-limit"],
		BurstLimit:     data["burst-limit"],
		BurstThreshold: data["burst-threshold"],
		BurstTime:      data["burst-time"],
		Priority:       data["priority"],
		Queue:          data["queue"],
		Comment:        data["comment"],
		Disabled:       pBool(data["disabled"]),
		Dynamic:        pBool(data["dynamic"]),
		Timestamp:      now,
	}
}

func (q *QueueSimple) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":              q.ID,
		"name":            q.Name,
		"target":          q.Target,
		"parent":          q.Parent,
		"packet_mark":     q.PacketMark,
		"rate_limit":      q.RateLimit,
		"max_limit":       q.MaxLimit,
		"burst_limit":     q.BurstLimit,
		"burst_threshold": q.BurstThreshold,
		"burst_time":      q.BurstTime,
		"priority":        q.Priority,
		"queue":           q.Queue,
		"comment":         q.Comment,
		"disabled":        fmt.Sprintf("%t", q.Disabled),
		"dynamic":         fmt.Sprintf("%t", q.Dynamic),
		"timestamp":       ts.Format(time.RFC3339),
	}
}
