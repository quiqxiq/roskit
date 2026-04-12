package domain

import "time"

// LogEntry represents a single RouterOS log entry.
type LogEntry struct {
	ID        string    // RouterOS internal identifier.
	Time      string    // Log timestamp as reported by the router.
	Topics    string    // Comma-separated log topics (e.g., "system,info").
	Message   string    // Log message content.
	Timestamp time.Time // When this entry was collected by roskit.
}

// ToTags returns InfluxDB-style tags for this log entry.
func (l *LogEntry) ToTags() map[string]string {
	return map[string]string{
		"topics": l.Topics,
	}
}

// ToFields returns InfluxDB-style fields for this log entry.
// Note: field named "log_time" instead of "time" because "time" is reserved in InfluxDB.
func (l *LogEntry) ToFields() map[string]interface{} {
	return map[string]interface{}{
		"log_time": l.Time,
		"message":  l.Message,
	}
}

// ToCacheData returns a flat key-value map for Redis HSET.
func (l *LogEntry) ToCacheData(ts time.Time) map[string]string {
	return map[string]string{
		"id":        l.ID,
		"log_time":  l.Time,
		"topics":    l.Topics,
		"message":   l.Message,
		"timestamp": ts.Format(time.RFC3339),
	}
}
