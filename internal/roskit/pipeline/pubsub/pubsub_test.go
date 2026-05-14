//go:build redis

package pubsub_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/pipeline/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipWithoutRedis(t *testing.T) {
	t.Helper()
	if os.Getenv("REDIS_ADDR") == "" && os.Getenv("REDIS_TEST") == "" {
		t.Skip("skipping: set REDIS_ADDR to run Redis tests")
	}
}

func redisConfig() pubsub.RedisConfig {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	return pubsub.RedisConfig{Addr: addr}
}

func newTestPublisher(t *testing.T) *pubsub.RedisPublisher {
	t.Helper()
	pub, err := pubsub.NewRedisPublisher(redisConfig(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pub.Close() })
	return pub
}

func newTestSubscriber(t *testing.T) *pubsub.RedisSubscriber {
	t.Helper()
	sub, err := pubsub.NewRedisSubscriber(redisConfig(), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sub.Close() })
	return sub
}

func TestRedisPublisher_Publish_Success(t *testing.T) {
	skipWithoutRedis(t)
	pub := newTestPublisher(t)

	err := pub.Publish(context.Background(), "test:channel:pub", pubsub.Message{
		Type:        "update",
		RouterID:    "r1",
		Measurement: "hotspot_user",
		EntityID:    "*1",
	})
	assert.NoError(t, err)
}

func TestRedisSubscriber_Subscribe_ReceivesMessage(t *testing.T) {
	skipWithoutRedis(t)
	pub := newTestPublisher(t)
	sub := newTestSubscriber(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	channel := "test:channel:sub:" + time.Now().Format("150405")
	ch, err := sub.Subscribe(ctx, channel)
	require.NoError(t, err)

	require.NoError(t, pub.Publish(context.Background(), channel, pubsub.Message{
		Type:        "update",
		RouterID:    "r1",
		Measurement: "hotspot_user",
		EntityID:    "*2",
	}))

	select {
	case received, ok := <-ch:
		require.True(t, ok, "channel should not be closed")
		assert.Equal(t, channel, received.Channel)
		assert.NotEmpty(t, received.Payload)
	case <-ctx.Done():
		t.Fatal("timed out waiting for message")
	}
}

func TestPubSub_Roundtrip(t *testing.T) {
	skipWithoutRedis(t)
	pub := newTestPublisher(t)
	sub := newTestSubscriber(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	channel := "test:roundtrip:" + time.Now().Format("150405.000")
	ch, err := sub.Subscribe(ctx, channel)
	require.NoError(t, err)

	require.NoError(t, pub.Publish(context.Background(), channel, pubsub.Message{
		Type:        "poll",
		RouterID:    "router-test",
		Measurement: "system_resource",
	}))

	select {
	case msg := <-ch:
		assert.Contains(t, string(msg.Payload), "router-test")
	case <-ctx.Done():
		t.Fatal("timed out waiting for roundtrip message")
	}
}

func TestRedisSubscriber_Cancel_ClosesChannel(t *testing.T) {
	skipWithoutRedis(t)
	sub := newTestSubscriber(t)

	ctx, cancel := context.WithCancel(context.Background())
	channel := "test:cancel:" + time.Now().Format("150405")

	ch, err := sub.Subscribe(ctx, channel)
	require.NoError(t, err)

	cancel()

	select {
	case _, ok := <-ch:
		assert.False(t, ok, "channel should be closed after ctx cancel")
	case <-time.After(2 * time.Second):
		t.Fatal("channel not closed after context cancellation")
	}
}
