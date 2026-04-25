package event

import (
	"context"
	"log/slog"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/cache"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/pubsub"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/timeseries"
)

const defaultCacheTTL = 5 * time.Minute

// realtimeMeasurements is the allowlist for pub-sub broadcast.
// Only these measurements are pushed to frontend subscribers in real time.
// Config/state data (hotspot_user, dhcp_lease, etc.) stays in Redis only.
var realtimeMeasurements = map[string]bool{
	"hotspot_active":    true,
	"interface_traffic": true,
	"system_resource":   true,
	"dhcp_lease":        true,
}

type Processor struct {
	cache      cache.Repository
	timeseries timeseries.Writer
	pubsub     pubsub.Publisher
	logger     *slog.Logger
}

func NewProcessor(
	c cache.Repository,
	ts timeseries.Writer,
	ps pubsub.Publisher,
	logger *slog.Logger,
) *Processor {
	if logger == nil {
		logger = slog.Default()
	}
	return &Processor{
		cache:      c,
		timeseries: ts,
		pubsub:     ps,
		logger:     logger,
	}
}

func (p *Processor) OnEvent(ctx context.Context, event behavior.StreamEvent) error {
	if event.IsDead {
		return p.handleDead(ctx, event)
	}
	return p.handleUpdate(ctx, event)
}

func (p *Processor) OnPoll(ctx context.Context, event behavior.PollEvent) error {
	if event.Err != nil {
		return nil
	}

	for _, row := range event.Rows {
		id := row[".id"]
		if id == "" {
			id = "singleton"
		}

		cacheKey := cache.FormatCacheKey(event.RouterID, event.Meta.Measurement, id)
		ttl := event.Meta.CacheTTL
		if ttl == 0 {
			ttl = defaultCacheTTL
		}

		if event.Meta.WriteTimeSeries {
			p.timeseries.WritePoint(ctx, timeseries.Point{
				Measurement: event.Meta.Measurement,
				Tags: map[string]string{
					"router_id": event.RouterID,
					"id":        id,
				},
				Fields:    toFloat64Fields(row),
				Timestamp: event.PollTime,
			})
		}

		if err := p.cache.SetSnapshot(ctx, cacheKey, row, ttl); err != nil {
			p.logger.Warn("processor: poll cache write failed",
				"key", cacheKey, "err", err)
		}
	}

	if realtimeMeasurements[event.Meta.Measurement] {
		channel := cache.FormatPubSubChannel(event.RouterID)
		p.pubsub.Publish(ctx, channel, pubsub.Message{
			Type:        "poll",
			RouterID:    event.RouterID,
			Measurement: event.Meta.Measurement,
			Timestamp:   event.PollTime,
		})
	}

	return nil
}

func (p *Processor) handleUpdate(ctx context.Context, event behavior.StreamEvent) error {
	meta := event.Meta
	fields := event.Fields

	id, _ := fields[".id"]
	if id == "" {
		id = "singleton"
	}

	cacheKey := cache.FormatCacheKey(event.RouterID, meta.Measurement, id)
	ttl := meta.CacheTTL
	if ttl == 0 {
		ttl = defaultCacheTTL
	}

	if meta.WriteTimeSeries {
		p.timeseries.WritePoint(ctx, timeseries.Point{
			Measurement: meta.Measurement,
			Tags: map[string]string{
				"router_id": event.RouterID,
				"id":        id,
				"category":  meta.Category,
			},
			Fields:    toFloat64Fields(fields),
			Timestamp: event.ReceivedAt,
		})
	}

	if err := p.cache.SetSnapshot(ctx, cacheKey, fields, ttl); err != nil {
		p.logger.Warn("processor: snapshot write failed", "key", cacheKey, "err", err)
	}

	if meta.IndexField != "" {
		if nameVal, ok := fields[meta.IndexField]; ok && nameVal != "" {
			indexKey := cache.FormatIndexKey(event.RouterID, meta.Measurement)
			if err := p.cache.SetIndex(ctx, indexKey, nameVal, cacheKey); err != nil {
				p.logger.Warn("processor: index write failed",
					"index_key", indexKey, "name", nameVal, "err", err)
			}
		}
	}

	if realtimeMeasurements[meta.Measurement] {
		channel := cache.FormatPubSubChannel(event.RouterID)
		p.pubsub.Publish(ctx, channel, pubsub.Message{
			Type:        "update",
			RouterID:    event.RouterID,
			Measurement: meta.Measurement,
			EntityID:    id,
			Fields:      fields,
			Timestamp:   event.ReceivedAt,
		})
	}

	return nil
}

func (p *Processor) handleDead(ctx context.Context, event behavior.StreamEvent) error {
	meta := event.Meta
	id, _ := event.Fields[".id"]
	if id == "" {
		return nil
	}

	cacheKey := cache.FormatCacheKey(event.RouterID, meta.Measurement, id)

	if meta.IndexField != "" {
		if nameVal, ok := event.Fields[meta.IndexField]; ok && nameVal != "" {
			indexKey := cache.FormatIndexKey(event.RouterID, meta.Measurement)
			if err := p.cache.DeleteIndex(ctx, indexKey, nameVal); err != nil {
				p.logger.Warn("processor: index delete failed", "err", err)
			}
		}
	}

	if err := p.cache.DeleteSnapshot(ctx, cacheKey); err != nil {
		p.logger.Warn("processor: snapshot delete failed", "key", cacheKey, "err", err)
	}

	if realtimeMeasurements[meta.Measurement] {
		channel := cache.FormatPubSubChannel(event.RouterID)
		p.pubsub.Publish(ctx, channel, pubsub.Message{
			Type:        "dead",
			RouterID:    event.RouterID,
			Measurement: meta.Measurement,
			EntityID:    id,
			Timestamp:   event.ReceivedAt,
		})
	}

	return nil
}

func toFloat64Fields(m map[string]string) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		if k == ".id" || k == ".dead" {
			continue
		}
		if k == "time" {
			k = "roskit_time"
		}
		out[k] = v
	}
	return out
}

var _ behavior.StreamSink = (*Processor)(nil)
var _ behavior.PollSink = (*Processor)(nil)
