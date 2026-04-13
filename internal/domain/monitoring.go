package domain

import "time"

// SystemHealth represents hardware health sensors (temperature, voltage, fan).
type SystemHealth struct {
	Name      string    // Sensor name (e.g., "cpu-temperature", "board-temperature1").
	Value     string    // Sensor value.
	Type      string    // Sensor type (e.g., "C", "V", "rpm").
	Timestamp time.Time // When this sample was collected.
}

func (h *SystemHealth) ToTags() map[string]string {
	return map[string]string{
		"name": h.Name,
		"type": h.Type,
	}
}

func (h *SystemHealth) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"value": h.Value,
	}
}

func (h *SystemHealth) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"name": h.Name, "value": h.Value, "type": h.Type,
		"timestamp": ts.Format(time.RFC3339),
	}
}

// WirelessRegistration represents a wireless client registration entry.
type WirelessRegistration struct {
	ID              string    // RouterOS internal identifier.
	Interface       string    // Wireless interface name.
	MacAddress      string    // Client MAC address.
	SignalStrength  string    // Signal strength (e.g., "-65").
	TxRate          string    // Transmit rate.
	RxRate          string    // Receive rate.
	Uptime          string    // Client connection uptime.
	LastActivity    string    // Time since last activity.
	Bytes           string    // Bytes transferred (rx/tx format).
	Packets         string    // Packets transferred.
	TxSignalStr     string    // TX signal strength.
	Timestamp       time.Time // When this sample was collected.
}

func (w *WirelessRegistration) ToTags() map[string]string {
	return map[string]string{
		"interface":   w.Interface,
		"mac_address": w.MacAddress,
	}
}

func (w *WirelessRegistration) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"signal_strength": w.SignalStrength,
		"tx_rate":         w.TxRate,
		"rx_rate":         w.RxRate,
		"uptime":          w.Uptime,
		"last_activity":   w.LastActivity,
		"bytes":           w.Bytes,
		"packets":         w.Packets,
	}
}

func (w *WirelessRegistration) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id": w.ID, "interface": w.Interface, "mac_address": w.MacAddress,
		"signal_strength": w.SignalStrength, "tx_rate": w.TxRate, "rx_rate": w.RxRate,
		"uptime": w.Uptime, "last_activity": w.LastActivity, "bytes": w.Bytes,
		"packets": w.Packets, "timestamp": ts.Format(time.RFC3339),
	}
}

// QueueTreeStats represents a queue tree rule with real-time statistics.
type QueueTreeStats struct {
	Name          string    // Queue rule name.
	Parent        string    // Parent queue name.
	PacketMark    string    // Packet mark filter.
	Rate          string    // Current rate (bps).
	PacketRate    string    // Current packet rate.
	QueuedBytes   string    // Bytes in queue.
	QueuedPackets string    // Packets in queue.
	Bytes         string    // Total bytes passed.
	Packets       string    // Total packets passed.
	Dropped       string    // Total dropped packets.
	MaxLimit      string    // Max bandwidth limit.
	LimitAt       string    // Guaranteed bandwidth.
	BurstLimit    string    // Burst limit.
	Timestamp     time.Time // When this sample was collected.
}

func (q *QueueTreeStats) ToTags() map[string]string {
	return map[string]string{
		"name":        q.Name,
		"parent":      q.Parent,
		"packet_mark": q.PacketMark,
	}
}

func (q *QueueTreeStats) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"rate":           q.Rate,
		"packet_rate":    q.PacketRate,
		"queued_bytes":   q.QueuedBytes,
		"queued_packets": q.QueuedPackets,
		"bytes":          q.Bytes,
		"packets":        q.Packets,
		"dropped":        q.Dropped,
		"max_limit":      q.MaxLimit,
		"limit_at":       q.LimitAt,
	}
}

func (q *QueueTreeStats) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"name": q.Name, "parent": q.Parent, "packet_mark": q.PacketMark,
		"rate": q.Rate, "packet_rate": q.PacketRate, "queued_bytes": q.QueuedBytes,
		"queued_packets": q.QueuedPackets, "bytes": q.Bytes, "packets": q.Packets,
		"dropped": q.Dropped, "max_limit": q.MaxLimit, "limit_at": q.LimitAt,
		"burst_limit": q.BurstLimit, "timestamp": ts.Format(time.RFC3339),
	}
}
