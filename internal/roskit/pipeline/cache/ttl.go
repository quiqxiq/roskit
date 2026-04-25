package cache

import "time"

const DefaultStreamTTL = 5 * time.Minute
const DefaultPollTTL = 2 * time.Minute
const DefaultQueryTTL = 10 * time.Second

func EffectiveTTL(requested time.Duration, fallback time.Duration) time.Duration {
	if requested > 0 {
		return requested
	}
	return fallback
}
