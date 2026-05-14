package pubsub

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

type RedisSubscriber struct {
	client *redis.Client
	logger *slog.Logger
}

func NewRedisSubscriber(cfg RedisConfig, logger *slog.Logger) (*RedisSubscriber, error) {
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
		return nil, fmt.Errorf("redis subscriber: connect %s: %w", cfg.Addr, err)
	}
	logger.Info("Redis subscriber initialized", "addr", cfg.Addr, "db", cfg.DB)
	return &RedisSubscriber{client: client, logger: logger}, nil
}

func (s *RedisSubscriber) Subscribe(ctx context.Context, channels ...string) (<-chan SubscriberMessage, error) {
	ps := s.client.Subscribe(ctx, channels...)

	if _, err := ps.Receive(ctx); err != nil {
		_ = ps.Close()
		return nil, fmt.Errorf("redis subscriber: SUBSCRIBE failed: %w", err)
	}

	out := make(chan SubscriberMessage, 64)

	go func() {
		defer close(out)
		defer ps.Close()

		redisCh := ps.Channel()
		for {
			select {
			case msg, ok := <-redisCh:
				if !ok {
					return
				}
				select {
				case out <- SubscriberMessage{Channel: msg.Channel, Payload: []byte(msg.Payload)}:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

func (s *RedisSubscriber) Close() error {
	return s.client.Close()
}

var _ Subscriber = (*RedisSubscriber)(nil)
