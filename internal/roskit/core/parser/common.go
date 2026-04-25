package parser

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func IsDead(pairs map[string]string) bool {
	v, ok := pairs[".dead"]
	if !ok {
		return false
	}
	return v == "true" || v == "yes"
}

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

func ParseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "yes"
}

func ParseMikroTikTime(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Now()
	}
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

func FormatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func FormatUint64(v uint64) string {
	return fmt.Sprintf("%d", v)
}

func FormatInt64(v int64) string {
	return fmt.Sprintf("%d", v)
}
