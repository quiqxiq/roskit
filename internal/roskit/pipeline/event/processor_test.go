//go:build redis

package event_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/behavior"
	"github.com/quiqxiq/roskit/internal/roskit/core/command"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/cache"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/event"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/pubsub"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/timeseries"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipWithoutRedis(t *testing.T) {
	t.Helper()
	if os.Getenv("REDIS_ADDR") == "" && os.Getenv("REDIS_TEST") == "" {
		t.Skip("skipping: set REDIS_ADDR to run Redis tests")
	}
}

func newTestCache(t *testing.T) *cache.RedisRepository {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	repo, err := cache.NewRedisRepository(cache.RedisConfig{Addr: addr, DB: 1}, slog.Default())
	require.NoError(t, err)
	t.Cleanup(func() { repo.Close() })
	return repo
}

func makeStreamEvent(routerID, measurement string, fields map[string]string, isDead bool) behavior.StreamEvent {
	return behavior.StreamEvent{
		Meta:       &command.CommandMeta{Measurement: measurement, IndexField: "name", CacheTTL: 5 * time.Minute},
		RouterID:   routerID,
		Fields:     fields,
		IsDead:     isDead,
		ReceivedAt: time.Now(),
	}
}

func TestProcessor_OnEvent_WriteCache(t *testing.T) {
	skipWithoutRedis(t)
	repo := newTestCache(t)
	ctx := context.Background()

	proc := event.NewProcessor(repo, timeseries.NoopWriter{}, pubsub.NoopPublisher{}, slog.Default())

	fields := map[string]string{".id": "*100", "name": "cache-test-user", "password": "secret"}
	evt := makeStreamEvent("proc-test-router", "hotspot_user", fields, false)
	require.NoError(t, proc.OnEvent(ctx, evt))

	key := cache.FormatCacheKey("proc-test-router", "hotspot_user", "*100")
	got, err := repo.GetSnapshot(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, fields, got)
}

func TestProcessor_OnEvent_WriteIndex(t *testing.T) {
	skipWithoutRedis(t)
	repo := newTestCache(t)
	ctx := context.Background()

	proc := event.NewProcessor(repo, timeseries.NoopWriter{}, pubsub.NoopPublisher{}, slog.Default())

	fields := map[string]string{".id": "*200", "name": "idx-test-user", "profile": "default"}
	evt := makeStreamEvent("proc-idx-router", "hotspot_user", fields, false)
	require.NoError(t, proc.OnEvent(ctx, evt))

	indexKey := cache.FormatIndexKey("proc-idx-router", "hotspot_user")
	got, err := repo.GetByIndex(ctx, indexKey, "idx-test-user")
	require.NoError(t, err)
	assert.Equal(t, fields, got)
}

func TestProcessor_OnEvent_PubSub_Noop(t *testing.T) {
	skipWithoutRedis(t)
	repo := newTestCache(t)
	ctx := context.Background()

	proc := event.NewProcessor(repo, timeseries.NoopWriter{}, pubsub.NoopPublisher{}, slog.Default())

	fields := map[string]string{".id": "*300", "name": "pubsub-user"}
	evt := makeStreamEvent("proc-pubsub-router", "hotspot_active", fields, false)
	require.NoError(t, proc.OnEvent(ctx, evt))
}

func TestProcessor_OnPoll_WriteCache(t *testing.T) {
	skipWithoutRedis(t)
	repo := newTestCache(t)
	ctx := context.Background()

	proc := event.NewProcessor(repo, timeseries.NoopWriter{}, pubsub.NoopPublisher{}, slog.Default())

	rows := []map[string]string{
		{".id": "*400", "name": "poll-user-1", "value": "10"},
		{".id": "*401", "name": "poll-user-2", "value": "20"},
	}
	evt := behavior.PollEvent{
		Meta:     &command.CommandMeta{Measurement: "hotspot_user", CacheTTL: 5 * time.Minute},
		RouterID: "proc-poll-router",
		Rows:     rows,
		PollTime: time.Now(),
	}
	require.NoError(t, proc.OnPoll(ctx, evt))

	key1 := cache.FormatCacheKey("proc-poll-router", "hotspot_user", "*400")
	got1, err := repo.GetSnapshot(ctx, key1)
	require.NoError(t, err)
	assert.Equal(t, rows[0], got1)

	key2 := cache.FormatCacheKey("proc-poll-router", "hotspot_user", "*401")
	got2, err := repo.GetSnapshot(ctx, key2)
	require.NoError(t, err)
	assert.Equal(t, rows[1], got2)
}

func TestProcessor_OnPoll_Error(t *testing.T) {
	skipWithoutRedis(t)
	repo := newTestCache(t)
	ctx := context.Background()

	proc := event.NewProcessor(repo, timeseries.NoopWriter{}, pubsub.NoopPublisher{}, slog.Default())

	evt := behavior.PollEvent{
		Meta:     &command.CommandMeta{Measurement: "hotspot_user", CacheTTL: 5 * time.Minute},
		RouterID: "proc-pollerr-router",
		Rows:     []map[string]string{{".id": "*500", "name": "should-not-exist"}},
		PollTime: time.Now(),
		Err:      assert.AnError,
	}
	require.NoError(t, proc.OnPoll(ctx, evt))

	key := cache.FormatCacheKey("proc-pollerr-router", "hotspot_user", "*500")
	got, err := repo.GetSnapshot(ctx, key)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestProcessor_HandleDead_DeleteCache(t *testing.T) {
	skipWithoutRedis(t)
	repo := newTestCache(t)
	ctx := context.Background()

	proc := event.NewProcessor(repo, timeseries.NoopWriter{}, pubsub.NoopPublisher{}, slog.Default())

	fields := map[string]string{".id": "*600", "name": "dead-cache-user"}
	evt := makeStreamEvent("proc-dead-router", "hotspot_user", fields, false)
	require.NoError(t, proc.OnEvent(ctx, evt))

	key := cache.FormatCacheKey("proc-dead-router", "hotspot_user", "*600")
	got, err := repo.GetSnapshot(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, fields, got)

	deadEvt := makeStreamEvent("proc-dead-router", "hotspot_user", fields, true)
	require.NoError(t, proc.OnEvent(ctx, deadEvt))

	got, err = repo.GetSnapshot(ctx, key)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestProcessor_HandleDead_DeleteIndex(t *testing.T) {
	skipWithoutRedis(t)
	repo := newTestCache(t)
	ctx := context.Background()

	proc := event.NewProcessor(repo, timeseries.NoopWriter{}, pubsub.NoopPublisher{}, slog.Default())

	fields := map[string]string{".id": "*700", "name": "dead-idx-user"}
	evt := makeStreamEvent("proc-deadidx-router", "hotspot_user", fields, false)
	require.NoError(t, proc.OnEvent(ctx, evt))

	indexKey := cache.FormatIndexKey("proc-deadidx-router", "hotspot_user")
	got, err := repo.GetByIndex(ctx, indexKey, "dead-idx-user")
	require.NoError(t, err)
	assert.Equal(t, fields, got)

	deadEvt := makeStreamEvent("proc-deadidx-router", "hotspot_user", fields, true)
	require.NoError(t, proc.OnEvent(ctx, deadEvt))

	got, err = repo.GetByIndex(ctx, indexKey, "dead-idx-user")
	require.NoError(t, err)
	assert.Nil(t, got)
}
