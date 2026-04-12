package domain

import "time"

// InterfaceStats represents real-time traffic statistics for a network interface.
type InterfaceStats struct {
	Name      string    // Interface name (e.g., "ether1", "wlan1").
	Type      string    // Interface type (e.g., "ether", "wlan", "bridge").
	RxByte    uint64    // Total received bytes counter.
	TxByte    uint64    // Total transmitted bytes counter.
	RxPacket  uint64    // Total received packets counter.
	TxPacket  uint64    // Total transmitted packets counter.
	RxDrop    uint64    // Total received dropped packets counter.
	TxDrop    uint64    // Total transmitted dropped packets counter.
	RxError   uint64    // Total received error packets counter.
	TxError   uint64    // Total transmitted error packets counter.
	Timestamp time.Time // When this sample was collected.
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
