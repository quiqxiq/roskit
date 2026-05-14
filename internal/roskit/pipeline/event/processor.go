package event

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/pubsub"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/timeseries"
)

var realtimeMeasurements = map[string]bool{
	"hotspot_active":    true,
	"hotspot_user":      true,
	"ip_binding":        true,
	"interface_traffic": true,
	"system_resource":   true,
	"dhcp_lease":        true,
	"ppp_secret":        true,
	"ppp_active":        true,
}

type Processor struct {
	timeseries timeseries.Writer
	pubsub     pubsub.Publisher
	logger     *slog.Logger
}

func NewProcessor(
	ts timeseries.Writer,
	ps pubsub.Publisher,
	logger *slog.Logger,
) *Processor {
	if logger == nil {
		logger = slog.Default()
	}
	return &Processor{
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
	}

	if realtimeMeasurements[event.Meta.Measurement] {
		channel := pubsub.FormatPubSubChannel(event.RouterID)
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

	if realtimeMeasurements[meta.Measurement] {
		channel := pubsub.FormatPubSubChannel(event.RouterID)
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

	if realtimeMeasurements[meta.Measurement] {
		channel := pubsub.FormatPubSubChannel(event.RouterID)
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
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			out[k] = n
		} else if f, err := strconv.ParseFloat(v, 64); err == nil {
			out[k] = f
		} else {
			out[k] = v
		}
	}
	return out
}

var _ behavior.StreamSink = (*Processor)(nil)
var _ behavior.PollSink = (*Processor)(nil)
