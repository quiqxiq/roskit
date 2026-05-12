package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type Cache struct {
	client *goredis.Client
	logger *slog.Logger
}

const (
	TTL30s            = 30 * time.Second
	TTL1m             = time.Minute
	TTL5m             = 5 * time.Minute
	TTL15m            = 15 * time.Minute
	TTL1h             = time.Hour
	TTLSalesDay       = 60 * time.Second
	TTLSalesMonth     = 5 * time.Minute
	TTLSalesPast      = 1 * time.Hour
	TTLDashboard      = 10 * time.Second
	TTLTemplates      = 5 * time.Minute
	TTLVoucherSession = 2 * time.Hour
)

func NewCache(client *goredis.Client) *Cache {
	return &Cache{
		client: client,
		logger: slog.Default().With("component", "appcache"),
	}
}

func NewCacheWithLogger(client *goredis.Client, logger *slog.Logger) *Cache {
	return &Cache{
		client: client,
		logger: logger.With("component", "appcache"),
	}
}

func SalesKey(routerID uint, period string) string {
	return fmt.Sprintf("mikhmon:sales:%d:%s", routerID, period)
}

func DashboardKey(routerID uint) string {
	return fmt.Sprintf("mikhmon:dashboard:%d", routerID)
}

func TemplateKey(routerID uint, name, part string) string {
	return fmt.Sprintf("mikhmon:template:%d:%s:%s", routerID, name, part)
}

func AuthRevokedKey(tokenID string) string {
	return fmt.Sprintf("mikhmon:auth:revoked:%s", tokenID)
}

func AuthRefreshKey(userID, tokenID string) string {
	return fmt.Sprintf("mikhmon:auth:refresh:%s:%s", userID, tokenID)
}

func VoucherSessionKey(routerID uint, gencode string) string {
	return fmt.Sprintf("mikhmon:vsession:%d:%s", routerID, gencode)
}

func HotspotUsersKey(routerID uint) string {
	return fmt.Sprintf("mikhmon:hotspot_users:%d", routerID)
}

const GlobalDashboardKey = "roskit:dashboard:global"

func (c *Cache) Get(ctx context.Context, key string) (string, bool, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == goredis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("cache get %s: %w", key, err)
	}
	return val, true, nil
}

func (c *Cache) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if ttl > 0 {
		return c.client.Set(ctx, key, value, ttl).Err()
	}
	return c.client.Set(ctx, key, value, 0).Err()
}

func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

func (c *Cache) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("cache setjson marshal: %w", err)
	}
	return c.Set(ctx, key, string(data), ttl)
}

func (c *Cache) GetJSON(ctx context.Context, key string, dest any) (bool, error) {
	val, found, err := c.Get(ctx, key)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false, fmt.Errorf("cache getjson unmarshal: %w", err)
	}
	return true, nil
}

func GetOrSetJSON[T any](c *Cache, ctx context.Context, key string, ttl time.Duration, fetch func() (T, error)) (T, error) {
	var zero T

	found, err := c.GetJSON(ctx, key, &zero)
	if err != nil {
		c.logger.WarnContext(ctx, "cache read failed, calling fetch directly", "key", key, "error", err)
		return fetch()
	}
	if found {
		return zero, nil
	}

	result, err := fetch()
	if err != nil {
		return zero, err
	}

	if setErr := c.SetJSON(ctx, key, result, ttl); setErr != nil {
		c.logger.WarnContext(ctx, "cache write failed, value not cached", "key", key, "error", setErr)
	}

	return result, nil
}

func (c *Cache) Invalidate(ctx context.Context, keys ...string) error {
	return c.Delete(ctx, keys...)
}

func (c *Cache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *Cache) Client() *goredis.Client {
	return c.client
}
