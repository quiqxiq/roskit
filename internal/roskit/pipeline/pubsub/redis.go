// Package pubsub defines the pub/sub contract for real-time event broadcasting.
package pubsub

import (
	"context"
	"encoding/json"
	"time"
)

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// Message is the payload broadcast to frontend subscribers.
// Replaces the ad-hoc string encoding used in the old repository.
type Message struct {
	Type        string            `json:"type"`        // update | dead | poll
	RouterID    string            `json:"router_id"`
	Measurement string            `json:"measurement"`
	EntityID    string            `json:"entity_id,omitempty"`
	Fields      map[string]string `json:"fields,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

// JSON serializes the message. Returns empty string on error.
func (m Message) JSON() string {
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// Publisher broadcasts events to a Redis pub/sub channel.
type Publisher interface {
	Publish(ctx context.Context, channel string, msg Message) error
	Close() error
}

// NoopPublisher discards all events.
type NoopPublisher struct{}

func (NoopPublisher) Publish(_ context.Context, _ string, _ Message) error { return nil }
func (NoopPublisher) Close() error                                           { return nil }

var _ Publisher = NoopPublisher{}

func FormatPubSubChannel(routerID string) string {
	return "roskit:telemetry:" + routerID
}

func FormatLogChannel(routerID, filter string) string {
	return "roskit:logs:" + routerID + ":" + filter
}