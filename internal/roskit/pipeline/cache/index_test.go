package cache_test

import (
	"testing"

	"github.com/quiqxiq/roskit/internal/roskit/pipeline/cache"
	"github.com/stretchr/testify/assert"
)

func TestFormatCacheKey(t *testing.T) {
	result := cache.FormatCacheKey("42", "hotspot_active", "100")
	assert.Equal(t, "roskit:42:hotspot_active:100", result)
}

func TestFormatIndexKey(t *testing.T) {
	result := cache.FormatIndexKey("42", "hotspot_active")
	assert.Equal(t, "roskit:42:idx:hotspot_active", result)
}

func TestFormatPubSubChannel(t *testing.T) {
	result := cache.FormatPubSubChannel("42")
	assert.Equal(t, "roskit:telemetry:42", result)
}

func TestFormatLogChannel(t *testing.T) {
	result := cache.FormatLogChannel("42", "error")
	assert.Equal(t, "roskit:logs:42:error", result)
}
