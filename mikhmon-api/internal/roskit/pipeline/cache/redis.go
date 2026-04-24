package cache

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type Repository interface {
	SetSnapshot(ctx context.Context, key string, data map[string]string, ttl time.Duration) error
	GetSnapshot(ctx context.Context, key string) (map[string]string, error)
	DeleteSnapshot(ctx context.Context, key string) error
	ScanByMeasurement(ctx context.Context, routerID, measurement string) ([]map[string]string, error)
	CountByMeasurement(ctx context.Context, routerID, measurement string) (int, error)
	SetIndex(ctx context.Context, indexKey, name, value string) error
	GetByIndex(ctx context.Context, indexKey, name string) (map[string]string, error)
	DeleteIndex(ctx context.Context, indexKey, name string) error
	RefreshTTL(ctx context.Context, key string, ttl time.Duration) error
	Close() error
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type RedisRepository struct {
	client *redis.Client
	logger *slog.Logger
}

func NewRedisRepository(cfg RedisConfig, logger *slog.Logger) (*RedisRepository, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis: connect %s: %w", cfg.Addr, err)
	}
	logger.Info("Redis repository initialized", "addr", cfg.Addr, "db", cfg.DB)
	return &RedisRepository{client: client, logger: logger}, nil
}

func (r *RedisRepository) SetSnapshot(ctx context.Context, key string, data map[string]string, ttl time.Duration) error {
	if len(data) == 0 {
		return nil
	}
	args := make([]interface{}, 0, len(data)*2)
	for k, v := range data {
		args = append(args, k, v)
	}
	pipe := r.client.Pipeline()
	pipe.HSet(ctx, key, args...)
	if ttl > 0 {
		pipe.Expire(ctx, key, ttl)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("redis: HSET %s: %w", key, err)
	}
	return nil
}

func (r *RedisRepository) GetSnapshot(ctx context.Context, key string) (map[string]string, error) {
	result, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis: HGETALL %s: %w", key, err)
	}
	return result, nil
}

func (r *RedisRepository) DeleteSnapshot(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis: DEL %s: %w", key, err)
	}
	return nil
}

func (r *RedisRepository) ScanByMeasurement(ctx context.Context, routerID, measurement string) ([]map[string]string, error) {
	pattern := fmt.Sprintf("roskit:%s:%s:*", routerID, measurement)
	var cursor uint64
	var results []map[string]string
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("redis: SCAN %s: %w", pattern, err)
		}
		cursor = nextCursor
		if len(keys) > 0 {
			pipe := r.client.Pipeline()
			cmds := make([]*redis.MapStringStringCmd, len(keys))
			for i, key := range keys {
				cmds[i] = pipe.HGetAll(ctx, key)
			}
			if _, err := pipe.Exec(ctx); err != nil {
				return nil, fmt.Errorf("redis: HGETALL batch %s: %w", pattern, err)
			}
			for _, cmd := range cmds {
				data := cmd.Val()
				if len(data) > 0 {
					results = append(results, data)
				}
			}
		}
		if cursor == 0 {
			break
		}
	}
	return results, nil
}

func (r *RedisRepository) CountByMeasurement(ctx context.Context, routerID, measurement string) (int, error) {
	pattern := fmt.Sprintf("roskit:%s:%s:*", routerID, measurement)
	var cursor uint64
	count := 0
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return 0, fmt.Errorf("redis: SCAN %s: %w", pattern, err)
		}
		cursor = nextCursor
		count += len(keys)
		if cursor == 0 {
			break
		}
	}
	return count, nil
}

func (r *RedisRepository) SetIndex(ctx context.Context, indexKey, name, value string) error {
	if name == "" {
		return nil
	}
	if err := r.client.HSet(ctx, indexKey, name, value).Err(); err != nil {
		return fmt.Errorf("redis: HSET index %s %s: %w", indexKey, name, err)
	}
	return nil
}

func (r *RedisRepository) GetByIndex(ctx context.Context, indexKey, name string) (map[string]string, error) {
	entityKey, err := r.client.HGet(ctx, indexKey, name).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis: HGET index %s %s: %w", indexKey, name, err)
	}
	return r.GetSnapshot(ctx, entityKey)
}

func (r *RedisRepository) DeleteIndex(ctx context.Context, indexKey, name string) error {
	if name == "" {
		return nil
	}
	if err := r.client.HDel(ctx, indexKey, name).Err(); err != nil {
		return fmt.Errorf("redis: HDEL index %s %s: %w", indexKey, name, err)
	}
	return nil
}

func (r *RedisRepository) RefreshTTL(ctx context.Context, key string, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = DefaultStreamTTL
	}
	if err := r.client.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("redis: EXPIRE %s: %w", key, err)
	}
	return nil
}

func (r *RedisRepository) Close() error {
	return r.client.Close()
}

type NoopRepository struct{}

func (NoopRepository) SetSnapshot(_ context.Context, _ string, _ map[string]string, _ time.Duration) error {
	return nil
}
func (NoopRepository) GetSnapshot(_ context.Context, _ string) (map[string]string, error) {
	return nil, nil
}
func (NoopRepository) DeleteSnapshot(_ context.Context, _ string) error { return nil }
func (NoopRepository) ScanByMeasurement(_ context.Context, _, _ string) ([]map[string]string, error) {
	return nil, nil
}
func (NoopRepository) CountByMeasurement(_ context.Context, _, _ string) (int, error) {
	return 0, nil
}
func (NoopRepository) SetIndex(_ context.Context, _, _, _ string) error { return nil }
func (NoopRepository) GetByIndex(_ context.Context, _, _ string) (map[string]string, error) {
	return nil, nil
}
func (NoopRepository) DeleteIndex(_ context.Context, _, _ string) error { return nil }
func (NoopRepository) RefreshTTL(_ context.Context, _ string, _ time.Duration) error {
	return nil
}
func (NoopRepository) Close() error { return nil }
