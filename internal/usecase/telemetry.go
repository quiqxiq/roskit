// Package usecase contains the business logic for processing telemetry events.
// It orchestrates data flow between the collector and repository layers
// without depending on any external library directly.
package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/quiqxiq/roskit/internal/domain"
	"github.com/quiqxiq/roskit/internal/repository"
	"github.com/quiqxiq/roskit/pkg/mikrotik"
)

// TelemetryUseCase orchestrates the processing of telemetry events.
// It receives events from the collector and routes them to the appropriate repositories.
// This is the central business logic component that ensures:
//   - Updated metrics are written to InfluxDB (time-series)
//   - Latest snapshots are cached in Redis (HSET)
//   - Real-time events are published via Redis Pub/Sub
//   - Dead entities are removed from the cache
type TelemetryUseCase struct {
	tsWriter  repository.TimeSeriesWriter
	cache     repository.CacheRepository
	publisher repository.PubSubPublisher
	logger    *slog.Logger
}

// NewTelemetryUseCase creates a new TelemetryUseCase with the given dependencies.
// All repositories are injected through interfaces, making this component
// easy to test with mocks.
func NewTelemetryUseCase(
	tsWriter repository.TimeSeriesWriter,
	cache repository.CacheRepository,
	publisher repository.PubSubPublisher,
	logger *slog.Logger,
) *TelemetryUseCase {
	return &TelemetryUseCase{
		tsWriter:  tsWriter,
		cache:     cache,
		publisher: publisher,
		logger:    logger,
	}
}

// ProcessEvent handles a telemetry event based on its type.
// For EventUpdate: writes to InfluxDB, updates Redis cache, publishes to Pub/Sub.
// For EventDead: removes the entity from Redis cache and publishes the dead event.
func (uc *TelemetryUseCase) ProcessEvent(ctx context.Context, event *domain.TelemetryEvent) error {
	if event == nil {
		return nil
	}

	switch event.Type {
	case domain.EventUpdate:
		return uc.handleUpdate(ctx, event)
	case domain.EventDead:
		return uc.handleDead(ctx, event)
	default:
		uc.logger.Warn("unknown event type", "type", event.Type, "router", event.RouterID)
		return nil
	}
}

// handleUpdate processes an entity update: write metrics, update cache, publish event.
func (uc *TelemetryUseCase) handleUpdate(ctx context.Context, event *domain.TelemetryEvent) error {
	// 1. Add router ID to tags for multi-router disambiguation.
	tags := make(map[string]string, len(event.Tags)+1)
	tags["router"] = event.RouterID
	for k, v := range event.Tags {
		tags[k] = v
	}

	// 2. Write to InfluxDB time-series.
	if err := uc.tsWriter.WritePoint(ctx, event.Measurement, tags, event.Fields, event.Timestamp); err != nil {
		uc.logger.Error("failed to write to time-series",
			"measurement", event.Measurement,
			"router", event.RouterID,
			"error", err,
		)
		// Don't return — continue to update cache and publish.
	}

	// 3. Update Redis cache snapshot.
	if event.CacheKey != "" && event.CacheData != nil {
		if err := uc.cache.SetSnapshot(ctx, event.CacheKey, event.CacheData); err != nil {
			uc.logger.Error("failed to update cache",
				"key", event.CacheKey,
				"error", err,
			)
		}
	}

	// 4. Publish to Redis Pub/Sub for real-time subscribers.
	channel := mikrotik.FormatPubSubChannel(event.RouterID)
	if err := uc.publishEvent(ctx, channel, event); err != nil {
		uc.logger.Error("failed to publish event",
			"channel", channel,
			"error", err,
		)
	}

	return nil
}

// handleDead processes an entity removal: delete from cache and publish dead event.
func (uc *TelemetryUseCase) handleDead(ctx context.Context, event *domain.TelemetryEvent) error {
	uc.logger.Info("entity removed from router",
		"router", event.RouterID,
		"measurement", event.Measurement,
		"cache_key", event.CacheKey,
	)

	// 1. Delete from Redis cache.
	if event.CacheKey != "" {
		if err := uc.cache.DeleteSnapshot(ctx, event.CacheKey); err != nil {
			uc.logger.Error("failed to delete cache entry",
				"key", event.CacheKey,
				"error", err,
			)
		}
	}

	// 2. Publish the dead event so subscribers can react (e.g., remove from dashboard).
	channel := mikrotik.FormatPubSubChannel(event.RouterID)
	if err := uc.publishEvent(ctx, channel, event); err != nil {
		uc.logger.Error("failed to publish dead event",
			"channel", channel,
			"error", err,
		)
	}

	return nil
}

// pubSubMessage is the JSON structure published to Redis Pub/Sub channels.
type pubSubMessage struct {
	RouterID    string                 `json:"router_id"`
	Measurement string                 `json:"measurement"`
	Type        string                 `json:"type"`
	Tags        map[string]string      `json:"tags,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	Timestamp   string                 `json:"timestamp"`
}

// publishEvent serializes a TelemetryEvent to JSON and publishes it to the given channel.
func (uc *TelemetryUseCase) publishEvent(ctx context.Context, channel string, event *domain.TelemetryEvent) error {
	msg := pubSubMessage{
		RouterID:    event.RouterID,
		Measurement: event.Measurement,
		Type:        event.Type.String(),
		Tags:        event.Tags,
		Fields:      event.Fields,
		Timestamp:   event.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	return uc.publisher.Publish(ctx, channel, data)
}
