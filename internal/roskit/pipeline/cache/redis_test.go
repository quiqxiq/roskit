//go:build redis

package cache_test

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/quiqxiq/roskit/internal/roskit/pipeline/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skipWithoutRedis(t *testing.T) {
	t.Helper()
	if os.Getenv("REDIS_ADDR") == "" && os.Getenv("REDIS_TEST") == "" {
		t.Skip("skipping: set REDIS_ADDR to run Redis tests")
	}
}

func newTestRepo(t *testing.T) *cache.RedisRepository {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	repo, err := cache.NewRedisRepository(cache.RedisConfig{Addr: addr}, slog.Default())
	require.NoError(t, err)
	t.Cleanup(func() { repo.Close() })
	return repo
}

func TestRedis_SetGetSnapshot(t *testing.T) {
	skipWithoutRedis(t)
	repo := newTestRepo(t)
	ctx := context.Background()

	key := cache.FormatCacheKey("test-router", "hotspot_user", "*1")
	data := map[string]string{".id": "*1", "name": "user1", "password": "pass1"}
	require.NoError(t, repo.SetSnapshot(ctx, key, data, 2*time.Minute))

	got, err := repo.GetSnapshot(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestRedis_DeleteSnapshot(t *testing.T) {
	skipWithoutRedis(t)
	repo := newTestRepo(t)
	ctx := context.Background()

	key := cache.FormatCacheKey("test-router", "hotspot_user", "*2")
	data := map[string]string{".id": "*2", "name": "user2"}
	require.NoError(t, repo.SetSnapshot(ctx, key, data, 2*time.Minute))
	require.NoError(t, repo.DeleteSnapshot(ctx, key))

	got, err := repo.GetSnapshot(ctx, key)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestRedis_ScanByMeasurement(t *testing.T) {
	skipWithoutRedis(t)
	repo := newTestRepo(t)
	ctx := context.Background()

	routerID := "test-scan-router"
	measurement := "hotspot_user_scan"
	for i := 0; i < 3; i++ {
		key := cache.FormatCacheKey(routerID, measurement, string(rune('A'+i)))
		data := map[string]string{".id": string(rune('A' + i)), "name": "user"}
		require.NoError(t, repo.SetSnapshot(ctx, key, data, 2*time.Minute))
	}

	results, err := repo.ScanByMeasurement(ctx, routerID, measurement)
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestRedis_CountByMeasurement(t *testing.T) {
	skipWithoutRedis(t)
	repo := newTestRepo(t)
	ctx := context.Background()

	routerID := "test-count-router"
	measurement := "hotspot_user_count"
	for i := 0; i < 3; i++ {
		key := cache.FormatCacheKey(routerID, measurement, string(rune('X'+i)))
		data := map[string]string{".id": string(rune('X' + i)), "val": "1"}
		require.NoError(t, repo.SetSnapshot(ctx, key, data, 2*time.Minute))
	}

	count, err := repo.CountByMeasurement(ctx, routerID, measurement)
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestRedis_DeleteByMeasurement(t *testing.T) {
	skipWithoutRedis(t)
	repo := newTestRepo(t)
	ctx := context.Background()

	routerID := "test-delmeas-router"
	measurement := "hotspot_user_delmeas"
	for i := 0; i < 3; i++ {
		key := cache.FormatCacheKey(routerID, measurement, string(rune('M'+i)))
		data := map[string]string{".id": string(rune('M' + i))}
		require.NoError(t, repo.SetSnapshot(ctx, key, data, 2*time.Minute))
	}

	require.NoError(t, repo.DeleteByMeasurement(ctx, routerID, measurement))

	results, err := repo.ScanByMeasurement(ctx, routerID, measurement)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestRedis_SetGetIndex(t *testing.T) {
	skipWithoutRedis(t)
	repo := newTestRepo(t)
	ctx := context.Background()

	entityKey := cache.FormatCacheKey("test-idx-router", "hotspot_user", "*10")
	data := map[string]string{".id": "*10", "name": "indexed-user"}
	require.NoError(t, repo.SetSnapshot(ctx, entityKey, data, 2*time.Minute))

	indexKey := cache.FormatIndexKey("test-idx-router", "hotspot_user")
	require.NoError(t, repo.SetIndex(ctx, indexKey, "indexed-user", entityKey))

	got, err := repo.GetByIndex(ctx, indexKey, "indexed-user")
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestRedis_DeleteIndex(t *testing.T) {
	skipWithoutRedis(t)
	repo := newTestRepo(t)
	ctx := context.Background()

	entityKey := cache.FormatCacheKey("test-delidx-router", "hotspot_user", "*11")
	data := map[string]string{".id": "*11", "name": "del-user"}
	require.NoError(t, repo.SetSnapshot(ctx, entityKey, data, 2*time.Minute))

	indexKey := cache.FormatIndexKey("test-delidx-router", "hotspot_user")
	require.NoError(t, repo.SetIndex(ctx, indexKey, "del-user", entityKey))
	require.NoError(t, repo.DeleteIndex(ctx, indexKey, "del-user"))

	got, err := repo.GetByIndex(ctx, indexKey, "del-user")
	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestRedis_RefreshTTL(t *testing.T) {
	skipWithoutRedis(t)
	repo := newTestRepo(t)
	ctx := context.Background()

	key := cache.FormatCacheKey("test-ttl-router", "hotspot_user", "*20")
	data := map[string]string{".id": "*20", "name": "ttl-user"}
	require.NoError(t, repo.SetSnapshot(ctx, key, data, 1*time.Second))

	require.NoError(t, repo.RefreshTTL(ctx, key, 10*time.Minute))

	time.Sleep(1500 * time.Millisecond)
	got, err := repo.GetSnapshot(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestNoopRepository(t *testing.T) {
	var repo cache.NoopRepository
	ctx := context.Background()

	assert.NoError(t, repo.SetSnapshot(ctx, "k", map[string]string{"a": "b"}, time.Minute))
	got, err := repo.GetSnapshot(ctx, "k")
	assert.NoError(t, err)
	assert.Nil(t, got)
	assert.NoError(t, repo.DeleteSnapshot(ctx, "k"))
	assert.NoError(t, repo.DeleteByMeasurement(ctx, "r", "m"))
	scan, err := repo.ScanByMeasurement(ctx, "r", "m")
	assert.NoError(t, err)
	assert.Nil(t, scan)
	count, err := repo.CountByMeasurement(ctx, "r", "m")
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.NoError(t, repo.SetIndex(ctx, "ik", "n", "v"))
	idx, err := repo.GetByIndex(ctx, "ik", "n")
	assert.NoError(t, err)
	assert.Nil(t, idx)
	assert.NoError(t, repo.DeleteIndex(ctx, "ik", "n"))
	assert.NoError(t, repo.RefreshTTL(ctx, "k", time.Minute))
	assert.NoError(t, repo.Close())
}
