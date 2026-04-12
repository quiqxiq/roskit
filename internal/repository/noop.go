package repository

import (
	"context"
	"time"
)

// ─── No-Op Implementations ──────────────────────────────────────────────────
// Use these when you want to disable a specific storage backend.
// For example, if you only need Redis caching and don't want InfluxDB,
// pass NoOpTimeSeriesWriter{} as the tsWriter parameter.

// NoOpTimeSeriesWriter is a no-op implementation of TimeSeriesWriter.
// Use this when you don't need time-series storage (InfluxDB).
type NoOpTimeSeriesWriter struct{}

func (n NoOpTimeSeriesWriter) WritePoint(_ context.Context, _ string, _ map[string]string, _ map[string]interface{}, _ time.Time) error {
	return nil
}
func (n NoOpTimeSeriesWriter) Flush(_ context.Context) error { return nil }
func (n NoOpTimeSeriesWriter) Close() error                  { return nil }

// NoOpCacheRepository is a no-op implementation of CacheRepository.
// Use this when you don't need snapshot caching (Redis HSET).
type NoOpCacheRepository struct{}

func (n NoOpCacheRepository) SetSnapshot(_ context.Context, _ string, _ map[string]string) error {
	return nil
}
func (n NoOpCacheRepository) GetSnapshot(_ context.Context, _ string) (map[string]string, error) {
	return nil, nil
}
func (n NoOpCacheRepository) DeleteSnapshot(_ context.Context, _ string) error { return nil }
func (n NoOpCacheRepository) Close() error                                     { return nil }

// NoOpPubSubPublisher is a no-op implementation of PubSubPublisher.
// Use this when you don't need real-time event broadcasting (Redis Pub/Sub).
type NoOpPubSubPublisher struct{}

func (n NoOpPubSubPublisher) Publish(_ context.Context, _ string, _ []byte) error { return nil }
func (n NoOpPubSubPublisher) Close() error                                        { return nil }
