package domain

import (
	"fmt"
	"time"
)

// QueueSimpleStats represents real-time statistics for a simple queue rule.
type QueueSimpleStats struct {
	Name              string    // Queue rule name (e.g., "TRAFIK").
	Target            string    // Target addresses (e.g., "192.168.230.0/24").
	Rate              string    // Current rate (e.g., "506.7kbps/20.0Mbps" upload/download).
	PacketRate        string    // Current packet rate (e.g., "956/1756").
	QueuedBytes       string    // Bytes currently in queue (e.g., "0/7200").
	QueuedPackets     string    // Packets currently in queue (e.g., "0/5").
	Bytes             string    // Total bytes passed (e.g., "2350082705/35903967571").
	Packets           string    // Total packets passed (e.g., "17349117/28163524").
	Dropped           string    // Total dropped packets (e.g., "6935/3063380").
	TotalRate         string    // Aggregated rate including children.
	TotalBytes        string    // Aggregated bytes including children.
	TotalPackets      string    // Aggregated packets including children.
	TotalDropped      string    // Aggregated dropped including children.
	TotalQueuedBytes  string    // Aggregated queued bytes including children.
	Timestamp         time.Time // When this sample was collected.
}

// ToTags returns InfluxDB-style tags for this queue.
func (q *QueueSimpleStats) ToTags() map[string]string {
	return map[string]string{
		"name":   q.Name,
		"target": q.Target,
	}
}

// ToFields returns InfluxDB-style fields for this queue.
func (q *QueueSimpleStats) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"rate":               q.Rate,
		"packet_rate":        q.PacketRate,
		"queued_bytes":       q.QueuedBytes,
		"queued_packets":     q.QueuedPackets,
		"bytes":              q.Bytes,
		"packets":            q.Packets,
		"dropped":            q.Dropped,
		"total_rate":         q.TotalRate,
		"total_bytes":        q.TotalBytes,
		"total_packets":      q.TotalPackets,
		"total_dropped":      q.TotalDropped,
		"total_queued_bytes": q.TotalQueuedBytes,
	}
}

// ToCacheData returns a flat key-value map for Redis HSET.
func (q *QueueSimpleStats) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"name":               q.Name,
		"target":             q.Target,
		"rate":               q.Rate,
		"packet_rate":        q.PacketRate,
		"queued_bytes":       q.QueuedBytes,
		"queued_packets":     q.QueuedPackets,
		"bytes":              q.Bytes,
		"packets":            q.Packets,
		"dropped":            q.Dropped,
		"total_rate":         q.TotalRate,
		"total_bytes":        fmt.Sprintf("%s", q.TotalBytes),
		"total_packets":      q.TotalPackets,
		"total_dropped":      q.TotalDropped,
		"total_queued_bytes": q.TotalQueuedBytes,
		"timestamp":          ts.Format(time.RFC3339),
	}
}
