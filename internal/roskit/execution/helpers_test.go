//go:build mikrotik

package execution

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"
)

func skipWithoutMikroTik(t *testing.T) {
	t.Helper()
	if os.Getenv("MIKROTIK_HOST") == "" {
		t.Skip("skipping: MIKROTIK_HOST not set")
	}
}

func loadRouterConfig() ConnConfig {
	host := os.Getenv("MIKROTIK_HOST")
	user := os.Getenv("MIKROTIK_USER")
	pass := os.Getenv("MIKROTIK_PASS")
	id := os.Getenv("MIKROTIK_ID")
	if id == "" {
		id = "test"
	}
	if host == "" {
		host = "192.168.233.1:8728"
	}
	if user == "" {
		user = "admin"
	}
	if pass == "" {
		pass = "r00t"
	}
	return ConnConfig{
		RouterID:       id,
		Address:        host,
		Username:       user,
		Password:       pass,
		DialTimeout: 10 * time.Second,
	}
}

func routerID() string {
	id := os.Getenv("MIKROTIK_ID")
	if id == "" {
		return "test"
	}
	return id
}

func uniqueName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano()%1000000)
}

func newTestPool(t *testing.T) (*Pool, func()) {
	t.Helper()
	cfg := loadRouterConfig()
	pool := NewPool(slog.Default())
	pool.Register(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	pool.Start(ctx)

	waitCtx, waitCancel := context.WithTimeout(ctx, 15*time.Second)
	defer waitCancel()
	if err := pool.WaitConnected(waitCtx, cfg.RouterID); err != nil {
		cancel()
		pool.Stop()
		t.Fatalf("pool did not connect: %v", err)
	}

	cleanup := func() {
		cancel()
		pool.Stop()
	}
	return pool, cleanup
}
