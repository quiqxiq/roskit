package cache

import "fmt"

func FormatCacheKey(routerID, measurement, entityID string) string {
	return fmt.Sprintf("roskit:%s:%s:%s", routerID, measurement, entityID)
}

func FormatIndexKey(routerID, measurement string) string {
	return fmt.Sprintf("roskit:%s:idx:%s", routerID, measurement)
}

func FormatPubSubChannel(routerID string) string {
	return fmt.Sprintf("roskit:telemetry:%s", routerID)
}

func FormatLogChannel(routerID, filter string) string {
	return fmt.Sprintf("roskit:logs:%s:%s", routerID, filter)
}
