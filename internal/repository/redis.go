package repository

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// RedisConfig holds configuration for the Redis connection.
type RedisConfig struct {
	// Addr is the Redis server address (e.g., "localhost:6380").
	Addr string
	// Password is the Redis password. Empty for no-auth setups.
	Password string
	// DB is the Redis database number (default: 0).
	DB int
}

// RedisRepository implements both CacheRepository and PubSubPublisher
// using a single Redis connection. It uses HSET for snapshot caching
// and PUBLISH for real-time event distribution.
type RedisRepository struct {
	client *redis.Client
	logger *slog.Logger
}

// NewRedisRepository creates a new Redis repository with the given configuration.
func NewRedisRepository(cfg RedisConfig, logger *slog.Logger) (*RedisRepository, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Verify connectivity.
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis: failed to connect to %s: %w", cfg.Addr, err)
	}

	logger.Info("Redis repository initialized", "addr", cfg.Addr, "db", cfg.DB)

	return &RedisRepository{
		client: client,
		logger: logger,
	}, nil
}

// SetSnapshot stores the latest state of a telemetry entity using HSET.
// All fields in the data map are stored as hash fields under the given key.
// This allows partial reads via HGET and full reads via HGETALL.
func (r *RedisRepository) SetSnapshot(ctx context.Context, key string, data map[string]string) error {
	if len(data) == 0 {
		return nil
	}

	// Convert map[string]string to []interface{} for HSET.
	args := make([]interface{}, 0, len(data)*2)
	for k, v := range data {
		args = append(args, k, v)
	}

	if err := r.client.HSet(ctx, key, args...).Err(); err != nil {
		return fmt.Errorf("redis: HSET %s failed: %w", key, err)
	}

	return nil
}

// GetSnapshot retrieves the latest cached state of a telemetry entity via HGETALL.
// Returns an empty map if the key does not exist.
func (r *RedisRepository) GetSnapshot(ctx context.Context, key string) (map[string]string, error) {
	result, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("redis: HGETALL %s failed: %w", key, err)
	}
	return result, nil
}

// DeleteSnapshot removes a cached entity from Redis.
// Called when .dead=yes is received from the MikroTik API, indicating
// the entity (e.g., a user session) no longer exists on the router.
func (r *RedisRepository) DeleteSnapshot(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis: DEL %s failed: %w", key, err)
	}

	r.logger.Debug("deleted cache snapshot", "key", key)
	return nil
}

// Publish sends a message to the specified Redis Pub/Sub channel.
// External subscribers listening on this channel will receive the message in real-time.
// Messages are fire-and-forget — if no subscriber is listening, the message is lost.
func (r *RedisRepository) Publish(ctx context.Context, channel string, message []byte) error {
	if err := r.client.Publish(ctx, channel, message).Err(); err != nil {
		return fmt.Errorf("redis: PUBLISH to %s failed: %w", channel, err)
	}
	return nil
}

// Close releases the Redis connection.
func (r *RedisRepository) Close() error {
	return r.client.Close()
}

// Client returns the underlying Redis client for advanced operations.
// Use this to subscribe to Pub/Sub channels from external consumers.
func (r *RedisRepository) Client() *redis.Client {
	return r.client
}
