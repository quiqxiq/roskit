// Package mikrotik provides utility functions for parsing MikroTik RouterOS
// API response values into Go native types.
package mikrotik

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// IsDead checks if a RouterOS API sentence contains the .dead attribute,
// indicating that the entity has been removed from the router.
func IsDead(pairs map[string]string) bool {
	v, ok := pairs[".dead"]
	if !ok {
		return false
	}
	return v == "true" || v == "yes"
}

// ParseUint64 safely converts a string to uint64, returning 0 on failure.
func ParseUint64(s string) uint64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// ParseInt64 safely converts a string to int64, returning 0 on failure.
func ParseInt64(s string) int64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// ParseBool converts MikroTik boolean strings to Go bool.
// Recognizes "true", "yes" as true; everything else as false.
func ParseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "yes"
}

// durationRegex matches MikroTik duration format components: weeks, days, hours, minutes, seconds.
var durationRegex = regexp.MustCompile(`(?:(\d+)w)?(?:(\d+)d)?(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?`)

// ParseDuration converts a MikroTik duration string (e.g., "1w2d3h4m5s", "3h25m", "45s")
// into a Go time.Duration. Returns 0 on empty or unparseable input.
func ParseDuration(s string) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	matches := durationRegex.FindStringSubmatch(s)
	if matches == nil {
		return 0
	}

	var d time.Duration
	if matches[1] != "" {
		weeks, _ := strconv.Atoi(matches[1])
		d += time.Duration(weeks) * 7 * 24 * time.Hour
	}
	if matches[2] != "" {
		days, _ := strconv.Atoi(matches[2])
		d += time.Duration(days) * 24 * time.Hour
	}
	if matches[3] != "" {
		hours, _ := strconv.Atoi(matches[3])
		d += time.Duration(hours) * time.Hour
	}
	if matches[4] != "" {
		minutes, _ := strconv.Atoi(matches[4])
		d += time.Duration(minutes) * time.Minute
	}
	if matches[5] != "" {
		seconds, _ := strconv.Atoi(matches[5])
		d += time.Duration(seconds) * time.Second
	}

	return d
}

// ParseMikroTikTime converts a MikroTik datetime string to Go time.Time.
// MikroTik uses formats like "jan/02/2006 15:04:05" or "2006-01-02 15:04:05".
// Returns current time if parsing fails.
func ParseMikroTikTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Now()
	}

	// Try common MikroTik datetime formats
	formats := []string{
		"jan/02/2006 15:04:05",
		"Jan/02/2006 15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t
		}
	}

	return time.Now()
}

// SentenceToMap converts a RouterOS API sentence (list of "=key=value" words)
// into a map. API attributes starting with "." (like ".id", ".dead") are also included.
func SentenceToMap(words []string) map[string]string {
	m := make(map[string]string, len(words))
	for _, word := range words {
		if len(word) == 0 {
			continue
		}
		// Handle =key=value format
		if word[0] == '=' {
			word = word[1:] // strip leading =
		}
		idx := strings.Index(word, "=")
		if idx < 0 {
			m[word] = ""
			continue
		}
		m[word[:idx]] = word[idx+1:]
	}
	return m
}

// FormatCacheKey creates a standardized Redis cache key for a telemetry entity.
// Format: roskit:{routerID}:{measurement}:{entityID}
func FormatCacheKey(routerID, measurement, entityID string) string {
	return fmt.Sprintf("roskit:%s:%s:%s", routerID, measurement, entityID)
}

// FormatPubSubChannel creates a standardized Redis Pub/Sub channel name.
// Format: roskit:telemetry:{routerID}
func FormatPubSubChannel(routerID string) string {
	return fmt.Sprintf("roskit:telemetry:%s", routerID)
}
