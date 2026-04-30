package mikrotik

import (
	"crypto/sha256"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var months = map[string]time.Month{
	"jan": time.January, "feb": time.February, "mar": time.March,
	"apr": time.April, "may": time.May, "jun": time.June,
	"jul": time.July, "aug": time.August, "sep": time.September,
	"oct": time.October, "nov": time.November, "dec": time.December,
}

func Parse(date, timeStr string, loc *time.Location) (time.Time, error) {
	if date == "" || timeStr == "" {
		return time.Time{}, fmt.Errorf("empty date or time")
	}
	if loc == nil {
		loc = time.UTC
	}

	parts := strings.SplitN(timeStr, ":", 3)
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid hour: %s", parts[0])
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid minute: %s", parts[1])
	}
	second, err := strconv.Atoi(parts[2])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid second: %s", parts[2])
	}

	dateParts := strings.SplitN(date, "/", 3)
	if len(dateParts) != 3 {
		return time.Time{}, fmt.Errorf("invalid date format: %s", date)
	}

	mon, ok := months[strings.ToLower(dateParts[0])]
	if !ok {
		return time.Time{}, fmt.Errorf("unknown month: %s", dateParts[0])
	}
	day, err := strconv.Atoi(dateParts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day: %s", dateParts[1])
	}
	year, err := strconv.Atoi(dateParts[2])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid year: %s", dateParts[2])
	}

	return time.Date(year, mon, day, hour, minute, second, 0, loc), nil
}

func ResolveLocation(tz string) *time.Location {
	if tz == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}

func MakeSaleIdempotencyKey(routerID uint, username string, soldAt time.Time) string {
	input := fmt.Sprintf("%d|%s|%s", routerID, username, soldAt.UTC().Format("2006-01-02T15:04:05Z"))
	h := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", h)
}
