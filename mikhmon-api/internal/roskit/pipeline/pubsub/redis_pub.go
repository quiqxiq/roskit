package pubsub

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"github.com/quiqxiq/roskit/internal/roskit/pipeline/cache"
)

type RedisPublisher struct {
	client *redis.Client
	logger *slog.Logger
}

func NewRedisPublisher(cfg cache.RedisConfig, logger *slog.Logger) (*RedisPublisher, error) {
	if logger == nil {
		logger = slog.Default()
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis publisher: connect %s: %w", cfg.Addr, err)
	}
	logger.Info("Redis publisher initialized", "addr", cfg.Addr, "db", cfg.DB)
	return &RedisPublisher{client: client, logger: logger}, nil
}

func (p *RedisPublisher) Publish(ctx context.Context, channel string, msg Message) error {
	payload := msg.JSON()
	if err := p.client.Publish(ctx, channel, payload).Err(); err != nil {
		p.logger.Warn("redis publisher: publish failed", "channel", channel, "err", err)
		return err
	}
	return nil
}

func (p *RedisPublisher) Close() error {
	return p.client.Close()
}

var _ Publisher = (*RedisPublisher)(nil)
